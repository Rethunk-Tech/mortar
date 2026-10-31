// Package devsvc starts and keeps the mods a player develops in the Develop tab: a new project from the game loader's
// template, wired into a development profile through an extra mods folder.
package devsvc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-cfgedit/msbuild"
	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
	"github.com/Rethunk-Tech/go-desktopenv/nowindow"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	maxTemplateZip  = 64 << 20
	toolTimeout     = 10 * time.Second
	cloneAttempts   = 10
	cloneRetryDelay = 2 * time.Second
)

var (
	modName = regexp.MustCompile(`^[A-Z][A-Za-z0-9]{1,63}$`)
	// codeName is an adopted C# project's name: its project file's, which need not follow the template's naming.
	codeName  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	repoName  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	ownerName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	refPath   = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
)

// Service creates and lists mod projects.
type Service struct {
	Home     string
	Settings *settings.Service
	Profiles *profile.Service
	HTTP     *http.Client
	// CacheDir keeps each template version Mortar downloaded and the template hive `dotnet new` runs in.
	CacheDir string
	// Run runs a tool (dotnet, git, gh) in dir and returns its output; nil runs it.
	Run func(ctx context.Context, dir, name string, args ...string) ([]byte, error)
	// Version is Mortar's version, which Nexus Mods and CurseForge are told in uploads.
	Version string
	// ThunderstoreURL, when set, replaces https://thunderstore.io/ for uploads (tests).
	ThunderstoreURL string
	// LicensesURL, when set, replaces GitHub's licenses API (tests).
	LicensesURL string
	// ArchiveURL, when set, replaces codeload.github.com for the template archive (tests).
	ArchiveURL func(t components.DevTemplate) string
	// LatestLoader, when set, replaces looking up the newest loader release (tests).
	LatestLoader func(ctx context.Context, gameID string) string

	codeMu sync.Mutex
	code   map[string]gameCode
}

// NewMod is what the Develop wizard asks for: the mod's name, the template's parameters by id (a multi-choice value
// joined with "|"), and where the project goes.
type NewMod struct {
	Game   string            `json:"game"`
	Name   string            `json:"name"`
	Values map[string]string `json:"values"`
	// Parent is the folder the project's folder, Repo, is made in.
	Parent string `json:"parent"`
	Repo   string `json:"repo"`
	// Profile is the development profile's id; empty creates "<game> Development".
	Profile string `json:"profile"`
	// CopyFrom is a profile whose mods a new development profile starts with; empty starts it empty.
	CopyFrom string `json:"copyFrom"`
	// License replaces the template's LICENSE with one of Licenses; empty keeps the template's own.
	License string `json:"license"`
	// GitHub creates Owner/Repo from the template with gh (Private or public) instead of copying its archive.
	GitHub  bool   `json:"github"`
	Owner   string `json:"owner"`
	Private bool   `json:"private"`
	// Kind is one of the game's AssetKinds for a mod made only of files; empty starts a code mod from the template.
	Kind string `json:"kind"`
}

// Tools is what building a mod needs on this computer.
type Tools struct {
	// DotnetSDKs are the installed .NET SDK versions; empty means none.
	DotnetSDKs []string `json:"dotnetSdks"`
	Git        bool     `json:"git"`
	// GitHubUser is the gh login, empty when gh is missing or signed out.
	GitHubUser string `json:"githubUser"`
	// GitHubOwners are the accounts a repository can be created under: the gh login, then its organisations.
	GitHubOwners []GitHubOwner `json:"githubOwners"`
	// ProjectsDir is where new projects go unless the player picked a folder before.
	ProjectsDir string `json:"projectsDir"`
	// GameDir is the install the project builds against; empty when the game is not found.
	GameDir string `json:"gameDir"`
}

// guardGit stops a project's own .git/config (from an archive or a foreign clone) from running programs: no fsmonitor, no hooks.
func guardGit(name string, args []string) []string {
	if name != "git" {
		return args
	}
	return append([]string{"-c", "core.fsmonitor=false", "-c", "core.hooksPath=" + os.DevNull}, args...)
}

// untrustedDir reports whether dir is the folder of a project the user has not trusted.
func (s *Service) untrustedDir(dir string) bool {
	for _, g := range s.Settings.Get().Games {
		if g == nil {
			continue
		}
		for _, p := range g.DevProjects {
			if p.Untrusted && filepath.Clean(p.Dir) == filepath.Clean(dir) {
				return true
			}
		}
	}
	return false
}

func (s *Service) run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if name == "git" && s.untrustedDir(dir) {
		// The repository's own .git/config can name a filter driver that runs on git commands.
		return nil, usererr.Wrap(usererr.Unconfirmed, fmt.Errorf("git does not run in %s until you trust it", dir))
	}
	if s.Run != nil {
		return s.Run(ctx, dir, name, args...)
	}
	args = guardGit(name, args)
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- fixed tools; arguments validated by the caller
	cmd.Dir = dir
	cmd.Env = hostenv.Environ()
	if name == "dotnet" {
		cmd.Env = s.dotnetEnv()
	}
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
	t.GitHubOwners = s.githubOwners(ctx)
	if len(t.GitHubOwners) > 0 {
		t.GitHubUser = t.GitHubOwners[0].Login
	}
	t.ProjectsDir = filepath.Join(s.Home, "Projects")
	if inst, err := game.ResolveInstall(s.Home, s.Settings.Get(), gameID, ""); err == nil {
		t.GameDir = inst.Dir
	}
	return t
}

// GitHubOwner is a GitHub account and its avatar's URL.
type GitHubOwner struct {
	Login  string `json:"login"`
	Avatar string `json:"avatar"`
}

const ownerFields = `.login+"\t"+.avatar_url`

func parseOwners(out []byte) []GitHubOwner {
	var owners []GitHubOwner
	for line := range strings.Lines(string(out)) {
		login, avatar, _ := strings.Cut(strings.TrimSpace(line), "\t")
		if ownerName.MatchString(login) {
			owners = append(owners, GitHubOwner{Login: login, Avatar: avatar})
		}
	}
	return owners
}

func (s *Service) githubOwners(ctx context.Context) []GitHubOwner {
	out, err := s.run(ctx, "", "gh", "api", "user", "--jq", ownerFields)
	if err != nil {
		return []GitHubOwner{}
	}
	owners := parseOwners(out)
	if orgs, err := s.run(ctx, "", "gh", "api", "--paginate", "user/orgs", "--jq", ".[]|"+ownerFields); err == nil {
		owners = append(owners, parseOwners(orgs)...)
	}
	return owners
}

// RepoCheck is what GitHub says of a new repository's place: the owner (nil when no such account) and whether the
// name is taken there.
type RepoCheck struct {
	Owner *GitHubOwner `json:"owner"`
	Taken bool         `json:"taken"`
}

func notFound(err error) bool { return err != nil && strings.Contains(err.Error(), "HTTP 404") }

// CheckRepo looks up owner on GitHub and whether it already has a repository named repo.
func (s *Service) CheckRepo(ctx context.Context, owner, repo string) (RepoCheck, error) {
	if !ownerName.MatchString(owner) || !repoName.MatchString(repo) {
		return RepoCheck{}, usererr.Wrap(usererr.Invalid, errors.New("not a GitHub account and repository name"))
	}
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()
	out, err := s.run(ctx, "", "gh", "api", "users/"+owner, "--jq", ownerFields)
	if notFound(err) {
		return RepoCheck{}, nil
	}
	if err != nil {
		return RepoCheck{}, err
	}
	owners := parseOwners(out)
	if len(owners) == 0 {
		return RepoCheck{}, nil
	}
	_, err = s.run(ctx, "", "gh", "api", "repos/"+owner+"/"+repo, "--jq", ".id")
	if err != nil && !notFound(err) {
		return RepoCheck{}, err
	}
	return RepoCheck{Owner: &owners[0], Taken: err == nil}, nil
}

// Projects lists the game's mod projects.
func (s *Service) Projects(gameID string) []settings.DevProject {
	return slices.Clone(s.Settings.Get().GamePrefs(gameID).DevProjects)
}

// Kinds are the asset kind of each of the game's projects by folder; a code project has none and is left out.
func (s *Service) Kinds(gameID string) map[string]string {
	out := map[string]string{}
	for _, p := range s.Projects(gameID) {
		if kind := assetKindOf(gameID, p.Dir); kind != "" {
			out[p.Dir] = kind
		}
	}
	return out
}

// TestMods lists the packages the profile's projects name to test with that the profile does not hold yet. A package
// one of the game's projects publishes is built from that project, so it is never missing.
func (s *Service) TestMods(gameID, profileID string) ([]string, error) {
	installed, err := s.Profiles.Installed(gameID, profileID)
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, im := range installed {
		if im.Source.Kind == profile.KindThunderstore {
			have[im.Source.Name] = true
		}
	}
	projects := s.Projects(gameID)
	for _, p := range projects {
		if info, err := projectInfo(p); err == nil && info.Package != "" {
			have[info.Package] = true
		}
	}
	missing := []string{}
	for _, p := range projects {
		if p.Profile != profileID {
			continue
		}
		for _, pkg := range readTestWith(p.Dir) {
			if !have[pkg] {
				missing = append(missing, pkg)
				have[pkg] = true
			}
		}
	}
	return missing, nil
}

// Packages lists the Thunderstore packages (Team-Name) the game's projects publish, so updates to them are not offered.
func (s *Service) Packages(gameID string) []string {
	var out []string
	for _, p := range s.Settings.Get().GamePrefs(gameID).DevProjects {
		if info, err := projectInfo(p); err == nil && info.Package != "" {
			out = append(out, info.Package)
		}
	}
	return out
}

// Forget drops the project in dir from the Develop tab; its files stay, and so does its last build in the profile.
func (s *Service) Forget(gameID, dir string) error { return s.Settings.ForgetDevProject(gameID, dir) }

// Move makes profileID the profile the project in dir installs into, as when its profile was deleted; its extra mods
// folder follows.
func (s *Service) Move(gameID, dir, profileID string) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	// A damaged profile is still one to install into once repaired.
	if _, err := s.Profiles.Get(gameID, profileID); err != nil && usererr.KindOf(err) != usererr.Damaged {
		return err
	}
	next := p
	next.Profile = profileID
	return s.Settings.ReplaceDevProject(gameID, p, next)
}

// Trash moves the project's folder to the system trash, where the file manager can restore it, takes its build out of
// its profile, and drops the project from the Develop tab. Only a folder the Develop tab lists is moved.
func (s *Service) Trash(gameID, dir string) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	if err := safefs.Trash(p.Dir); err != nil {
		return err
	}
	if err := s.dropBuild(gameID, p); err != nil {
		return err
	}
	return s.Forget(gameID, p.Dir)
}

func (m NewMod) check() error {
	bad := func(what string) error { return usererr.Wrap(usererr.Invalid, errors.New(what)) }
	switch {
	case !modName.MatchString(m.Name):
		return bad("the mod name must start with a capital letter and hold only letters and digits")
	case !repoName.MatchString(m.Repo):
		return bad("the folder name holds only letters, digits, dots, dashes and underscores")
	case !filepath.IsAbs(m.Parent):
		return bad("pick the folder the project goes in")
	case m.GitHub && !ownerName.MatchString(m.Owner):
		return bad("name the GitHub account or organisation")
	}
	return nil
}

// TemplateForm is what the wizard asks for the game's template.
func (s *Service) TemplateForm(ctx context.Context, gameID string) (Form, error) {
	tpl, ok := Template(gameID)
	if !ok {
		return Form{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("%s has no mod template", gameID))
	}
	dir, err := s.templateDir(ctx, tpl)
	if err != nil {
		return Form{}, err
	}
	form, err := ReadForm(dir)
	form.Recommends = append([]components.DevRecommend{}, tpl.Recommends...)
	if folders := contentFolders(gameID); len(folders) > 0 {
		form.Recommends = append(form.Recommends, contentRecommends(folders)...)
		form.Parameters = append(form.Parameters, contentQuestion(folders, false))
	}
	return form, err
}

// Create starts a mod project from the game's template with `dotnet new`, points it at the game, records the values
// it used, and makes its build output an extra mods folder of the development profile (created when none is picked).
// Nothing is pushed after a GitHub repository is created.
func (s *Service) Create(ctx context.Context, m NewMod) (_ settings.DevProject, err error) {
	if err := m.check(); err != nil {
		return settings.DevProject{}, err
	}
	if m.Kind != "" {
		return s.createAsset(ctx, m)
	}
	tpl, ok := Template(m.Game)
	if !ok {
		return settings.DevProject{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("%s has no mod template", m.Game))
	}
	raw, err := s.templateDir(ctx, tpl)
	if err != nil {
		return settings.DevProject{}, err
	}
	form, err := ReadForm(raw)
	if err != nil {
		return settings.DevProject{}, err
	}
	ticked, err := tickedContent(m.Game, m.Values, false)
	if err != nil {
		return settings.DevProject{}, err
	}
	dest, url, err := m.destination()
	if err != nil {
		return settings.DevProject{}, err
	}
	defer removeIfFailed(dest, &err)
	args, used, err := form.args(m.Values, map[string]string{fillRepositoryURL: url, fillLoaderPackVersion: s.loaderPackVersion(ctx, m.Game)})
	if err != nil {
		return settings.DevProject{}, err
	}
	if err := s.instantiate(ctx, tpl, raw, form, m, dest, args); err != nil {
		return settings.DevProject{}, err
	}
	if err := record(dest, m.Game, tpl, m.Name, used); err != nil {
		return settings.DevProject{}, err
	}
	if err := writeFiles(dest, contentStarters(codeContentRoot, m.Name, ticked)); err != nil {
		return settings.DevProject{}, err
	}
	if err := writeUserProps(dest, s.Tools(ctx, m.Game).GameDir); err != nil {
		return settings.DevProject{}, err
	}
	if err := s.applyLicense(ctx, dest, m.License); err != nil {
		return settings.DevProject{}, err
	}
	s.commit(ctx, dest, m.GitHub, m.Name)
	return s.enroll(m.Game, dest, m.Name, m.Profile, m.CopyFrom, false, false)
}

// instantiate runs `dotnet new` on the template in raw, in Mortar's own template hive so nothing is installed for
// the player's own `dotnet new`. A GitHub copy is generated beside the clone and replaces its files.
func (s *Service) instantiate(ctx context.Context, tpl components.DevTemplate, raw string, form Form, m NewMod, dest string, args []string) (err error) {
	defer removeIfFailed(dest, &err)
	hive := filepath.Join(s.CacheDir, "dev-template-hive")
	if _, err := s.run(ctx, "", "dotnet", "new", "install", raw, "--force", "--debug:custom-hive", hive); err != nil {
		if _, missing := errors.AsType[*exec.Error](err); missing {
			return usererr.Wrap(usererr.NotFound, errors.New("install the .NET SDK to start a mod"))
		}
		return err
	}
	out := dest
	if m.GitHub {
		if err := s.fromGitHub(ctx, tpl, m, dest); err != nil {
			return err
		}
		out = filepath.Join(m.Parent, ".mortar-new-"+m.Repo)
		defer func() { _ = safefs.RemoveAll(out) }()
	}
	newArgs := append([]string{"new", form.shortName, "--debug:custom-hive", hive, "-n", m.Name, "-o", out}, args...)
	if _, err := s.run(ctx, "", "dotnet", newArgs...); err != nil {
		return err
	}
	if m.GitHub {
		return replaceTree(dest, out)
	}
	return nil
}

// removeIfFailed deletes dest, a folder this call made, when the call failed: a retry meets no "already exists". A
// GitHub repository made by then stays.
func removeIfFailed(dest string, err *error) {
	if *err != nil {
		_ = safefs.RemoveAll(dest)
	}
}

// replaceTree swaps everything in dest but its .git for what generated holds.
func replaceTree(dest, generated string) error {
	old, err := os.ReadDir(dest)
	if err != nil {
		return err
	}
	for _, e := range old {
		if e.Name() != ".git" {
			if err := safefs.RemoveAll(filepath.Join(dest, e.Name())); err != nil {
				return err
			}
		}
	}
	made, err := os.ReadDir(generated)
	if err != nil {
		return err
	}
	for _, e := range made {
		if err := safefs.Rename(filepath.Join(generated, e.Name()), filepath.Join(dest, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// projectRecord is .mortar/template.json in a project: the template it started from and the values it was filled
// with, so a newer template can be filled the same way.
type projectRecord struct {
	Template string            `json:"template"`
	Ref      string            `json:"ref"`
	Name     string            `json:"name"`
	Values   map[string]string `json:"values"`
	// Game is the catalog id of the game the project is for.
	Game string `json:"game,omitempty"`
	// TemplateIcon is the SHA-256 of the template's icon.png, so Pack can say the mod still wears it.
	TemplateIcon string `json:"templateIcon,omitempty"`
	// TestWith are packages (Team-Name) that Test installs into the project's profile beside it.
	TestWith []string `json:"testWith,omitempty"`
	siteIDs
}

const templateRecordFile = "template.json"

// readTemplateRecord reads the project's .mortar/template.json; false when it is missing or damaged.
func readTemplateRecord(dir string) (projectRecord, bool) {
	body, err := safefs.ReadFile(filepath.Join(dir, ".mortar", templateRecordFile))
	var rec projectRecord
	return rec, err == nil && json.Unmarshal(body, &rec) == nil
}

func writeTemplateRecord(dir string, rec projectRecord) error {
	body, err := recordFile(rec)
	if err != nil {
		return err
	}
	return writeTracked(dir, templateRecordFile, body)
}

func record(dir, gameID string, tpl components.DevTemplate, name string, values map[string]string) error {
	rec := projectRecord{Template: tpl.Repo, Ref: tpl.Ref, Name: name, Values: values, Game: gameID}
	if icon, err := safefs.ReadFile(filepath.Join(dir, "icon.png")); err == nil {
		rec.TemplateIcon = fmt.Sprintf("%x", sha256.Sum256(icon))
	}
	return writeTemplateRecord(dir, rec)
}

// templateIcon is the hash of the icon the template gave the project: recorded with it, else (for a project made
// before that) in Mortar's settings.
func templateIcon(p settings.DevProject) string {
	if rec, ok := readTemplateRecord(p.Dir); ok && rec.TemplateIcon != "" {
		return rec.TemplateIcon
	}
	return p.TemplateIcon
}

// recordedGame is the game the project says it is for, empty when it names none.
func recordedGame(dir string) string {
	if rec, ok := readRawRecord(dir); ok && rec.Game != "" {
		return rec.Game
	}
	rec, _ := readTemplateRecord(dir)
	return rec.Game
}

// readTestWith is the packages Test installs beside the project in dir.
func readTestWith(dir string) []string {
	if rec, ok := readRawRecord(dir); ok {
		return rec.TestWith
	}
	rec, _ := readTemplateRecord(dir)
	return rec.TestWith
}

// writeTestWith records the packages Test installs beside the project in dir, in the record the project keeps.
func writeTestWith(dir string, ids []string) error {
	if rec, ok := readRawRecord(dir); ok {
		rec.TestWith = ids
		return writeRecord(dir, rec)
	}
	rec, ok := readTemplateRecord(dir)
	if !ok && len(ids) == 0 {
		return nil
	}
	rec.TestWith = ids
	return writeTemplateRecord(dir, rec)
}

// loaderPackVersion is the newest release of the game's loader, empty when it cannot be looked up (the template's
// own version stands).
func (s *Service) loaderPackVersion(ctx context.Context, gameID string) string {
	if s.LatestLoader != nil {
		return s.LatestLoader(ctx, gameID)
	}
	l, ok := game.PrimaryLoader(gameID)
	r, isReleases := l.(loader.Releases)
	info, known := components.Game(gameID)
	if !ok || !isReleases || !known {
		return ""
	}
	v, err := r.Latest(ctx, info)
	if err != nil {
		return ""
	}
	return v
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
		_ = safefs.RemoveAll(dest)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(cloneRetryDelay):
		}
	}
	return fmt.Errorf("cloning %s: %w", full, err)
}

// TemplateEnv names a template to use instead of the catalog's, for trying one before it is published: a local
// template folder, a local .zip of one, or an archive URL. A local one is read again on every use, so edits show.
const TemplateEnv = "MORTAR_DEV_TEMPLATE"

// templateDir is the template at tpl's tag, downloaded once into the cache: the same files for the wizard's
// questions and for `dotnet new`.
func (s *Service) templateDir(ctx context.Context, tpl components.DevTemplate) (string, error) {
	url := "https://codeload.github.com/" + tpl.Repo + "/zip/refs/tags/" + tpl.Ref
	override := os.Getenv(TemplateEnv)
	switch {
	case s.ArchiveURL != nil:
		url = s.ArchiveURL(tpl)
	case strings.Contains(override, "://"):
		url = override
	case override != "":
		return s.localTemplate(override)
	}
	dir := filepath.Join(s.CacheDir, "dev-templates", refPath.ReplaceAllString(tpl.Repo+"@"+tpl.Ref, "_"))
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(templateFile))); err == nil {
		return dir, nil
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
		return "", err
	}
	work, err := os.MkdirTemp(filepath.Dir(dir), ".download-")
	if err != nil {
		return "", err
	}
	defer func() { _ = safefs.RemoveAll(work) }()
	zip := filepath.Join(work, "template.zip")
	if err := github.Download(ctx, s.HTTP, url, zip, maxTemplateZip, nil); err != nil {
		return "", fmt.Errorf("downloading the template %s %s: %w", tpl.Repo, tpl.Ref, err)
	}
	unpacked := filepath.Join(work, "files")
	if err := os.Mkdir(unpacked, 0o750); err != nil {
		return "", err
	}
	if err := archive.Extract(zip, unpacked); err != nil {
		return "", err
	}
	// GitHub's archive holds one folder, <repo>-<ref>.
	root := unpacked
	if entries, err := os.ReadDir(unpacked); err == nil && len(entries) == 1 && entries[0].IsDir() {
		root = filepath.Join(unpacked, entries[0].Name())
	}
	_ = safefs.RemoveAll(dir)
	return dir, safefs.Rename(root, dir)
}

// localTemplate copies a template folder, or unpacks a .zip of one, into the cache fresh.
func (s *Service) localTemplate(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := safefs.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%s names no template: %w", TemplateEnv, err)
	}
	dir := filepath.Join(s.CacheDir, "dev-templates", "local")
	if err := safefs.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	if info.IsDir() {
		err = datadir.CopyTree(abs, dir)
		// The template's git history isn't part of the template.
		err = errors.Join(err, safefs.RemoveAll(filepath.Join(dir, ".git")))
	} else {
		err = archive.Extract(abs, dir)
		// A GitHub-style archive holds one folder.
		if entries, readErr := os.ReadDir(dir); err == nil && readErr == nil && len(entries) == 1 && entries[0].IsDir() {
			inner := filepath.Join(dir, entries[0].Name())
			moved := dir + ".unpacked"
			err = errors.Join(safefs.RemoveAll(moved), safefs.Rename(inner, moved), safefs.RemoveAll(dir), safefs.Rename(moved, dir))
		}
	}
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(templateFile))); err != nil {
		return "", fmt.Errorf("%s (%s) is not a template: it has no %s", TemplateEnv, abs, templateFile)
	}
	log.Printf("dev template: using the local template copied to %s", dir)
	return dir, nil
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

// copyMods starts the new development profile with the mods the player chose in fromID, and what they need; local
// mods stay behind, since another project's build output is bound to its own profile.
func (s *Service) copyMods(gameID, fromID, toID string) error {
	installed, err := s.Profiles.Installed(gameID, fromID)
	if err != nil {
		return err
	}
	var ids []mod.ID
	for _, im := range installed {
		if im.Source.Kind != profile.KindLocal && !im.Implicit {
			ids = append(ids, im.ModID())
		}
	}
	if len(ids) == 0 {
		return nil
	}
	_, err = s.Profiles.CopyModsWithNeeds(gameID, fromID, toID, ids)
	return err
}

func writeUserProps(dir, gameDir string) error {
	if gameDir == "" {
		return nil
	}
	if err := msbuild.CheckValue(gameDir); err != nil {
		return fmt.Errorf("the game folder %w", err)
	}
	body := "<Project>\n  <PropertyGroup>\n    <GameRootDir>" + msbuild.Escape(filepath.Clean(gameDir)+string(filepath.Separator)) + "</GameRootDir>\n  </PropertyGroup>\n</Project>\n"
	return safefs.WriteFile(filepath.Join(dir, "Config.Build.user.props"), []byte(body), 0o600)
}
