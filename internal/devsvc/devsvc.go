// Package devsvc starts and keeps the mods a player develops in the Develop tab: a new project from the game loader's
// template, wired into a development profile through an extra mods folder.
package devsvc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/nowindow"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	maxTemplateZip  = 64 << 20
	maxFilledFile   = 1 << 20
	toolTimeout     = 10 * time.Second
	maxDescription  = 250
	cloneAttempts   = 10
	cloneRetryDelay = 2 * time.Second
)

var (
	modName    = regexp.MustCompile(`^[A-Z][A-Za-z0-9]{1,63}$`)
	teamName   = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)
	guidPrefix = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)*$`)
	repoName   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	ownerName  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
)

// Service creates and lists mod projects.
type Service struct {
	Home     string
	Settings *settings.Service
	Profiles *profile.Service
	HTTP     *http.Client
	// Run runs a tool (dotnet, git, gh) in dir and returns its output; nil runs it.
	Run func(ctx context.Context, dir, name string, args ...string) ([]byte, error)
	// ArchiveURL, when set, replaces codeload.github.com for the template archive (tests).
	ArchiveURL func(t components.DevTemplate) string
}

// NewMod is what the Develop wizard asks for.
type NewMod struct {
	Game        string `json:"game"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Team        string `json:"team"`
	GUIDPrefix  string `json:"guidPrefix"`
	Description string `json:"description"`
	// Parent is the folder the project's folder, Repo, is made in.
	Parent string `json:"parent"`
	Repo   string `json:"repo"`
	// Profile is the development profile's id; empty creates "<game> Development".
	Profile string `json:"profile"`
	// GitHub creates Owner/Repo from the template with gh (Private or public) instead of copying its archive.
	GitHub  bool   `json:"github"`
	Owner   string `json:"owner"`
	Private bool   `json:"private"`
}

// Tools is what building a mod needs on this computer.
type Tools struct {
	// DotnetSDKs are the installed .NET SDK versions; empty means none.
	DotnetSDKs []string `json:"dotnetSdks"`
	Git        bool     `json:"git"`
	// GitHubUser is the gh login, empty when gh is missing or signed out.
	GitHubUser string `json:"githubUser"`
	// GameDir is the install the project builds against; empty when the game is not found.
	GameDir string `json:"gameDir"`
}

func (s *Service) run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if s.Run != nil {
		return s.Run(ctx, dir, name, args...)
	}
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- fixed tools; arguments validated by the caller
	cmd.Dir = dir
	nowindow.Set(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %s: %w: %s", name, args[0], err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// Template is the template a new mod for the game starts from, if its loader names one.
func Template(gameID string) (components.DevTemplate, bool) {
	info, ok := components.Game(gameID)
	if !ok {
		return components.DevTemplate{}, false
	}
	for _, l := range info.Loaders {
		if l.DevTemplate != nil {
			return *l.DevTemplate, true
		}
	}
	return components.DevTemplate{}, false
}

// Tools reports the .NET SDKs, git, the gh login and the game folder a new project builds against.
func (s *Service) Tools(ctx context.Context, gameID string) Tools {
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()
	t := Tools{DotnetSDKs: []string{}}
	if out, err := s.run(ctx, "", "dotnet", "--list-sdks"); err == nil {
		for line := range strings.Lines(string(out)) {
			if v, _, ok := strings.Cut(strings.TrimSpace(line), " "); ok {
				t.DotnetSDKs = append(t.DotnetSDKs, v)
			}
		}
	}
	_, err := s.run(ctx, "", "git", "--version")
	t.Git = err == nil
	if out, err := s.run(ctx, "", "gh", "api", "user", "--jq", ".login"); err == nil {
		t.GitHubUser = strings.TrimSpace(string(out))
	}
	if inst, err := game.ResolveInstall(s.Home, s.Settings.Get(), gameID, ""); err == nil {
		t.GameDir = inst.Dir
	}
	return t
}

// Projects lists the game's mod projects.
func (s *Service) Projects(gameID string) []settings.DevProject {
	return slices.Clone(s.Settings.Get().GamePrefs(gameID).DevProjects)
}

// Forget drops the project in dir from the Develop tab; its files stay.
func (s *Service) Forget(gameID, dir string) error { return s.Settings.ForgetDevProject(gameID, dir) }

func (m NewMod) check() error {
	bad := func(what string) error { return usererr.Wrap(usererr.Invalid, errors.New(what)) }
	switch {
	case !modName.MatchString(m.Name):
		return bad("the mod name must start with a capital letter and hold only letters and digits")
	case !teamName.MatchString(m.Team):
		return bad("the Thunderstore team holds only letters, digits and underscores")
	case !guidPrefix.MatchString(m.GUIDPrefix):
		return bad("the id prefix holds lower-case letters and digits between dots, such as com.yourname")
	case strings.TrimSpace(m.Author) == "":
		return bad("name the author")
	case strings.TrimSpace(m.Description) == "" || len([]rune(m.Description)) > maxDescription:
		return bad(fmt.Sprintf("describe the mod in one line of at most %d characters", maxDescription))
	case !repoName.MatchString(m.Repo) || m.Repo == "." || m.Repo == "..":
		return bad("the folder name holds only letters, digits, dots, dashes and underscores")
	case !filepath.IsAbs(m.Parent):
		return bad("pick the folder the project goes in")
	case m.GitHub && !ownerName.MatchString(m.Owner):
		return bad("name the GitHub account or organisation")
	}
	return nil
}

// Create starts a mod project from the game's template, fills its placeholders, points it at the game, and makes its
// build output an extra mods folder of the development profile (created when none is picked). Nothing is pushed.
func (s *Service) Create(ctx context.Context, m NewMod) (settings.DevProject, error) {
	if err := m.check(); err != nil {
		return settings.DevProject{}, err
	}
	tpl, ok := Template(m.Game)
	if !ok {
		return settings.DevProject{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("%s has no mod template", m.Game))
	}
	dest := filepath.Join(m.Parent, m.Repo)
	if _, err := os.Lstat(dest); err == nil {
		return settings.DevProject{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("%s already exists", dest))
	}
	url := ""
	if m.GitHub {
		if err := s.fromGitHub(ctx, tpl, m, dest); err != nil {
			return settings.DevProject{}, err
		}
		url = "https://github.com/" + m.Owner + "/" + m.Repo
	} else if err := s.fromArchive(ctx, tpl, dest); err != nil {
		return settings.DevProject{}, err
	}
	if err := Fill(dest, Values{Name: m.Name, Author: m.Author, Team: m.Team, GUIDPrefix: m.GUIDPrefix, Description: m.Description, RepositoryURL: url, Year: time.Now().Year()}); err != nil {
		return settings.DevProject{}, err
	}
	gameDir := s.Tools(ctx, m.Game).GameDir
	if err := writeUserProps(dest, gameDir); err != nil {
		return settings.DevProject{}, err
	}
	s.commit(ctx, dest, m.GitHub, m.Name)
	profileID := m.Profile
	if profileID == "" {
		name := m.Game
		if info, ok := components.Game(m.Game); ok {
			name = info.Name
		}
		p, err := s.Profiles.Create(m.Game, name+" Development")
		if err != nil {
			return settings.DevProject{}, err
		}
		profileID = p.ID
	}
	// The template builds an unpacked package here; it exists before the first build so the folder watch can start.
	output := filepath.Join(dest, "artifacts", "bin", m.Name, "debug")
	if err := os.MkdirAll(output, 0o750); err != nil {
		return settings.DevProject{}, err
	}
	p := settings.DevProject{Dir: dest, Name: m.Name, Profile: profileID, Output: output}
	return p, s.Settings.AddDevProject(m.Game, p)
}

func (s *Service) fromGitHub(ctx context.Context, tpl components.DevTemplate, m NewMod, dest string) error {
	visibility := "--public"
	if m.Private {
		visibility = "--private"
	}
	full := m.Owner + "/" + m.Repo
	if _, err := s.run(ctx, "", "gh", "repo", "create", full, "--template", tpl.Repo, visibility); err != nil {
		return err
	}
	// GitHub fills a repo from its template a moment after creating it.
	var err error
	for range cloneAttempts {
		if _, err = s.run(ctx, m.Parent, "gh", "repo", "clone", full, m.Repo); err == nil {
			if entries, _ := os.ReadDir(dest); len(entries) > 1 {
				return nil
			}
		}
		_ = fsx.RemoveAll(dest)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(cloneRetryDelay):
		}
	}
	return fmt.Errorf("cloning %s: %w", full, err)
}

func (s *Service) fromArchive(ctx context.Context, tpl components.DevTemplate, dest string) error {
	url := "https://codeload.github.com/" + tpl.Repo + "/zip/refs/tags/" + tpl.Ref
	if s.ArchiveURL != nil {
		url = s.ArchiveURL(tpl)
	} else if o := archiveOverride(); o != "" {
		url = o
	}
	work, err := os.MkdirTemp(filepath.Dir(dest), ".mortar-template-")
	if err != nil {
		return err
	}
	defer func() { _ = fsx.RemoveAll(work) }()
	zip := filepath.Join(work, "template.zip")
	if err := github.Download(ctx, s.HTTP, url, zip, maxTemplateZip, nil); err != nil {
		return fmt.Errorf("downloading the template %s %s: %w", tpl.Repo, tpl.Ref, err)
	}
	unpacked := filepath.Join(work, "files")
	if err := os.Mkdir(unpacked, 0o750); err != nil {
		return err
	}
	if err := archive.Extract(zip, unpacked); err != nil {
		return err
	}
	// GitHub's archive holds one folder, <repo>-<ref>.
	root := unpacked
	if entries, err := os.ReadDir(unpacked); err == nil && len(entries) == 1 && entries[0].IsDir() {
		root = filepath.Join(unpacked, entries[0].Name())
	}
	return os.Rename(root, dest)
}

// commit records the filled template as the project's first commit (after the template's own, for a GitHub copy),
// when git is there and knows who the player is; without either the files are still ready.
func (s *Service) commit(ctx context.Context, dir string, cloned bool, name string) {
	if !cloned {
		if _, err := s.run(ctx, dir, "git", "init", "-q", "-b", "main"); err != nil {
			return
		}
	}
	if _, err := s.run(ctx, dir, "git", "add", "-A"); err != nil {
		return
	}
	_, _ = s.run(ctx, dir, "git", "commit", "-q", "-m", "Start "+name)
}

func writeUserProps(dir, gameDir string) error {
	if gameDir == "" {
		return nil
	}
	body := "<Project>\n  <PropertyGroup>\n    <GameRootDir>" + xmlEscape(filepath.Clean(gameDir)+string(filepath.Separator)) + "</GameRootDir>\n  </PropertyGroup>\n</Project>\n"
	return fsx.WriteFile(filepath.Join(dir, "Config.Build.user.props"), []byte(body), 0o600)
}

// Values fill a template's placeholders.
type Values struct {
	Name, Author, Team, GUIDPrefix, Description, RepositoryURL string
	Year                                                       int
}

// Fill replaces the template placeholders in every text file below dir (outside .git) and renames the files and
// folders whose names hold ModTemplate. Author and description are XML-escaped: the project files are XML, and
// Markdown shows the entities as the characters.
func Fill(dir string, v Values) error {
	r := strings.NewReplacer(
		"ModRepositoryUrl", xmlEscape(v.RepositoryURL),
		"ModDescription", xmlEscape(v.Description),
		"ModTemplate", v.Name,
		"ModAuthor", xmlEscape(v.Author),
		"ModTeam", v.Team,
		"modauthor", v.GUIDPrefix,
		"ModYear", strconv.Itoa(v.Year),
	)
	// Rooted, so a link inside the template cannot lead a read, write or rename outside the project.
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	var renames []string
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}
		name := filepath.FromSlash(path)
		if path != "." && strings.Contains(d.Name(), "ModTemplate") {
			renames = append(renames, name)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFilledFile {
			return err
		}
		body, err := root.ReadFile(name)
		if err != nil || bytes.IndexByte(body, 0) >= 0 {
			return err
		}
		if filled := r.Replace(string(body)); filled != string(body) {
			return root.WriteFile(name, []byte(filled), info.Mode().Perm())
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Deepest first, so a folder moves after the files in it.
	slices.Reverse(renames)
	for _, path := range renames {
		next := filepath.Join(filepath.Dir(path), strings.ReplaceAll(filepath.Base(path), "ModTemplate", v.Name))
		if err := root.Rename(path, next); err != nil {
			return err
		}
	}
	return nil
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
