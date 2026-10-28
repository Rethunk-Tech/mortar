package devsvc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/modpic"
	"github.com/Rethunk-Tech/mortar/internal/msbuild"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// The project's own files are the truth about it: Mortar reads them each time and follows a rename made outside it.
const propsFile = "Directory.Build.props"

var (
	categorySlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	teamPattern  = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9_]{0,62}[A-Za-z0-9])?$`)
	guidPattern  = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)*$`)
)

// ProjectInfo is what a project's files say about the mod, read fresh each time.
type ProjectInfo struct {
	// Missing is set when the project's folder or its project file is gone; nothing else is.
	Missing  bool   `json:"missing"`
	Name     string `json:"name"`
	GUID     string `json:"guid"`
	IDPrefix string `json:"idPrefix"`
	Author   string `json:"author"`
	Team     string `json:"team"`
	// Package is the Thunderstore package (Team-Name) the project publishes, empty without a team.
	Package string `json:"package"`
	// Version is the version the project builds, from Directory.Build.props.
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
	// Released is the latest vX.Y.Z tag's version, empty before the first release.
	Released string `json:"released"`
	// Icon is the project's icon.png as a small data URL, empty when the project has none.
	Icon string `json:"icon"`
	// Git is the working tree's state, nil when the project is not a git repository or git is missing.
	Git *GitState `json:"git"`
	// BuiltAt is when the debug build's output was written, in Unix ms; 0 before the first build.
	BuiltAt int64 `json:"builtAt"`
	// Stale is set when a source or asset file changed after that build.
	Stale bool `json:"stale"`
	// ClientOnly is set when the template's audience says only the player who installs the mod needs it, so testing it
	// needs no second copy.
	ClientOnly bool `json:"clientOnly"`
	// Kind is the catalog asset kind of a project made only of files; empty for code.
	Kind string `json:"kind"`
}

// GitState is what a project card shows of its repository.
type GitState struct {
	Branch string `json:"branch"`
	// Changed counts the files with uncommitted changes, untracked ones included.
	Changed int `json:"changed"`
	// Unpushed counts the commits no remote has.
	Unpushed int `json:"unpushed"`
}

// ProjectEdit is a change to a project's details; a new Name renames the project.
type ProjectEdit struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Team        string   `json:"team"`
	IDPrefix    string   `json:"idPrefix"`
	Categories  []string `json:"categories"`
}

// setProp sets a property of an MSBuild file's text, adding it when missing; a file that does not parse stays as it is.
func setProp(body, name, value string) string {
	out, err := msbuild.SetProperty(body, name, value)
	if err != nil {
		return body
	}
	return out
}

// csprojOf is the project's one project file under src/, which names it; false when the project is gone.
func csprojOf(dir string) (string, bool) {
	found, _ := filepath.Glob(filepath.Join(dir, "src", "*", "*.csproj"))
	if len(found) == 0 {
		return "", false
	}
	return found[0], true
}

// Inspect reads the project's details from its files. When the project was renamed outside Mortar, the record and the
// extra mods folder follow the new build output, and the install from the old one leaves its profile.
func (s *Service) Inspect(ctx context.Context, gameID, dir string) (ProjectInfo, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return ProjectInfo{}, err
	}
	if rec, ok := readAssetRecord(p.Dir); ok && p.Kind != "" {
		info := assetInfo(p.Dir, rec)
		info.Released = s.latestRelease(ctx, p.Dir)
		info.Icon = iconData(p.Dir)
		info.Git = s.gitState(ctx, p.Dir)
		return info, nil
	}
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		// Moved or deleted outside Mortar: the card offers to remove it.
		return ProjectInfo{Missing: true, Name: p.Name, Categories: []string{}}, nil
	}
	info, err := readInfo(p.Dir, csproj)
	if err != nil {
		return ProjectInfo{}, err
	}
	info.Released = s.latestRelease(ctx, p.Dir)
	info.Icon = iconData(p.Dir)
	info.Git = s.gitState(ctx, p.Dir)
	info.BuiltAt, info.Stale = buildState(p)
	if body, err := fsx.ReadFile(filepath.Join(p.Dir, ".mortar", "template.json")); err == nil {
		var rec projectRecord
		info.ClientOnly = json.Unmarshal(body, &rec) == nil && rec.Values["audience"] == "client"
	}
	base := strings.TrimSuffix(filepath.Base(csproj), ".csproj")
	if base != p.Name {
		if err := s.follow(gameID, p, base); err != nil {
			return ProjectInfo{}, err
		}
	}
	return info, nil
}

// buildState reads the last build from the files, so it survives a restart: the newest DLL in the output, and whether
// anything under src/ (build folders aside) or assets/ is newer.
func buildState(p settings.DevProject) (int64, bool) {
	dlls, _ := filepath.Glob(filepath.Join(p.Output, "*.dll"))
	var built time.Time
	for _, d := range dlls {
		if fi, err := os.Stat(d); err == nil && fi.ModTime().After(built) {
			built = fi.ModTime()
		}
	}
	if built.IsZero() {
		return 0, false
	}
	stale := false
	for _, root := range []string{filepath.Join(p.Dir, "src"), filepath.Join(p.Dir, "assets")} {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || stale {
				return filepath.SkipDir
			}
			if d.IsDir() {
				if n := d.Name(); n == "bin" || n == "obj" {
					return filepath.SkipDir
				}
				return nil
			}
			if fi, err := d.Info(); err == nil && fi.ModTime().After(built) {
				stale = true
				return filepath.SkipAll
			}
			return nil
		})
	}
	return built.UnixMilli(), stale
}

// iconPx is the size the card's icon is shrunk to: twice what it draws, so it stays sharp on a dense screen.
const iconPx = 96

// iconData is the project's icon.png, shrunk, as a data URL; a project without one, or with a broken one, has none.
func iconData(dir string) string {
	b, err := fsx.ReadFile(filepath.Join(dir, "icon.png"))
	if err != nil {
		return ""
	}
	small, typ, err := modpic.Shrink(b, "image/png", iconPx)
	if err != nil {
		return ""
	}
	return "data:" + typ + ";base64," + base64.StdEncoding.EncodeToString(small)
}

func readInfo(dir, csproj string) (ProjectInfo, error) {
	project, err := fsx.ReadFile(csproj)
	if err != nil {
		return ProjectInfo{}, err
	}
	props, _ := fsx.ReadFile(filepath.Join(dir, propsFile))
	base := strings.TrimSuffix(filepath.Base(csproj), ".csproj")
	info := ProjectInfo{
		Name:        cmpOr(msbuild.Property(string(project), "AssemblyTitle"), base),
		GUID:        msbuild.Property(string(project), "AssemblyName"),
		Author:      msbuild.Property(string(props), "Authors"),
		Team:        msbuild.Property(string(props), "ThunderstoreTeam"),
		Description: msbuild.Property(string(project), "ThunderstoreDescription"),
		Categories:  []string{},
	}
	info.IDPrefix = strings.TrimSuffix(info.GUID, "."+info.Name)
	info.Package = packageOf(dir)
	info.Version = msbuild.Property(string(props), "Version")
	for c := range strings.SplitSeq(msbuild.Property(string(project), "ThunderstoreCategories"), ";") {
		if c = strings.TrimSpace(c); c != "" {
			info.Categories = append(info.Categories, c)
		}
	}
	return info, nil
}

// follow points the record and the extra mods folder at the build output of the project file named name, and takes
// the install from the old output out of the development profile.
func (s *Service) follow(gameID string, p settings.DevProject, name string) error {
	next := p
	next.Name = name
	next.Output = filepath.Join(p.Dir, "artifacts", "bin", name, "debug")
	if err := os.MkdirAll(next.Output, 0o750); err != nil {
		return err
	}
	if err := s.Settings.ReplaceDevProject(gameID, p, next); err != nil {
		return err
	}
	return s.dropBuild(gameID, p)
}

// dropBuild takes the install of the project's build output out of its development profile.
func (s *Service) dropBuild(gameID string, p settings.DevProject) error {
	profiles, err := s.Profiles.List(gameID)
	if err != nil {
		return err
	}
	for _, pr := range profiles {
		if pr.ID != p.Profile {
			continue
		}
		var stale []string
		for _, e := range pr.Entries {
			if e.Source.Folder == p.Output {
				stale = append(stale, e.Key)
			}
		}
		if len(stale) > 0 {
			_, err = s.Profiles.RemoveEntries(gameID, pr.ID, stale)
		}
	}
	return err
}

func (e ProjectEdit) check() error {
	bad := func(what string) error { return usererr.Wrap(usererr.Invalid, errors.New(what)) }
	oneLine := func(v string) bool { return strings.TrimSpace(v) != "" && !strings.ContainsAny(v, "\r\n") }
	switch {
	case !modName.MatchString(e.Name):
		return bad("the mod name must start with a capital letter and hold only letters and digits")
	case !oneLine(e.Description) || len([]rune(e.Description)) > maxDescription:
		return bad(fmt.Sprintf("describe the mod in one line of at most %d characters", maxDescription))
	case !oneLine(e.Author):
		return bad("name the author")
	case !teamPattern.MatchString(e.Team):
		return bad("the team holds only letters, digits and underscores")
	case !guidPattern.MatchString(e.IDPrefix):
		return bad("the id prefix holds lower-case letters and digits between dots, such as com.yourname")
	case slices.ContainsFunc(e.Categories, func(c string) bool { return !categorySlug.MatchString(c) }):
		return bad("categories are Thunderstore category slugs")
	}
	return nil
}

// Edit writes new details into the project's files, renaming the project first when the name changed.
func (s *Service) Edit(ctx context.Context, gameID, dir string, e ProjectEdit) (ProjectInfo, error) {
	if err := e.check(); err != nil {
		return ProjectInfo{}, err
	}
	p, err := s.project(gameID, dir)
	if err != nil {
		return ProjectInfo{}, err
	}
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		return ProjectInfo{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("%s has no project file under src", p.Dir))
	}
	old, err := readInfo(p.Dir, csproj)
	if err != nil {
		return ProjectInfo{}, err
	}
	if e.Name != old.Name {
		if csproj, err = s.rename(ctx, p.Dir, csproj, old.Name, e.Name); err != nil {
			return ProjectInfo{}, err
		}
	}
	if err := editFile(csproj, func(body string) string {
		for name, v := range map[string]string{
			"AssemblyName": e.IDPrefix + "." + e.Name, "AssemblyTitle": e.Name, "RootNamespace": e.Name,
			"ThunderstoreName": e.Name, "ThunderstoreDescription": e.Description,
		} {
			body = setProp(body, name, v)
		}
		// A ; is the list separator MSBuild splits ThunderPipe's string[] on, so only the XML escape applies.
		if out, err := msbuild.SetRawProperty(body, "ThunderstoreCategories", msbuild.EscapeXML(strings.Join(e.Categories, ";"))); err == nil {
			body = out
		}
		return body
	}); err != nil {
		return ProjectInfo{}, err
	}
	if err := editFile(filepath.Join(p.Dir, propsFile), func(body string) string {
		body = setProp(body, "ThunderstoreTeam", e.Team)
		body = setProp(body, "Authors", e.Author)
		body = setProp(body, "Company", e.Author)
		if c := msbuild.Property(body, "Copyright"); old.Author != "" && strings.HasSuffix(c, old.Author) {
			body = setProp(body, "Copyright", strings.TrimSuffix(c, old.Author)+e.Author)
		}
		return body
	}); err != nil {
		return ProjectInfo{}, err
	}
	if old.Team != e.Team {
		_ = editFile(filepath.Join(p.Dir, "DEVELOPING.md"), func(body string) string {
			// The keyring commands name the team as the item's username, in its Windows target and in its label; a
			// longer team that starts with the old one is another team.
			return regexp.MustCompile(`(username |thunderstore:|Thunderstore )`+regexp.QuoteMeta(old.Team)+`\b`).
				ReplaceAllString(body, "${1}"+strings.ReplaceAll(e.Team, "$", "$$"))
		})
	}
	if old.Author != "" && old.Author != e.Author {
		_ = editFile(filepath.Join(p.Dir, "LICENSE"), func(body string) string {
			return regexp.MustCompile(`(Copyright \(c\) \d{4} )`+regexp.QuoteMeta(old.Author)).ReplaceAllString(body, "${1}"+e.Author)
		})
	}
	return s.Inspect(ctx, gameID, dir)
}

func editFile(path string, fn func(string) string) error {
	body, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	if next := fn(string(body)); next != string(body) {
		return fsx.WriteFile(path, []byte(next), 0o644) // #nosec G306 -- a tracked project file
	}
	return nil
}

// rename moves the project from old to name: its folder, project and solution files (with git when the project is a
// repository, so history follows), the namespace in its code, its README heading and its bundle's name.
func (s *Service) rename(ctx context.Context, dir, csproj, old, name string) (string, error) {
	move := func(from, to string) error {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			if _, err := s.run(ctx, dir, "git", "mv", "--", from, to); err == nil {
				return nil
			}
		}
		return fsx.Rename(filepath.Join(dir, from), filepath.Join(dir, to))
	}
	srcOld, srcNew := filepath.Join("src", filepath.Base(filepath.Dir(csproj))), filepath.Join("src", name)
	if _, err := os.Lstat(filepath.Join(dir, srcNew)); err == nil {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%s already exists", srcNew))
	}
	if err := move(srcOld, srcNew); err != nil {
		return "", err
	}
	if err := move(filepath.Join(srcNew, filepath.Base(csproj)), filepath.Join(srcNew, name+".csproj")); err != nil {
		return "", err
	}
	if slnx, _ := filepath.Glob(filepath.Join(dir, "*.slnx")); len(slnx) == 1 {
		if err := move(filepath.Base(slnx[0]), name+".slnx"); err != nil {
			return "", err
		}
		oldPath := filepath.ToSlash(srcOld) + "/" + filepath.Base(csproj)
		_ = editFile(filepath.Join(dir, name+".slnx"), func(b string) string {
			out, err := msbuild.SetAttr(b, func(e msbuild.Elem, _ string) bool {
				return e.Name == "Project" && e.Attrs["Path"] == oldPath
			}, "Path", "src/"+name+"/"+name+".csproj")
			if err != nil {
				return b
			}
			return out
		})
	}
	word := regexp.MustCompile(`\b` + regexp.QuoteMeta(old) + `\b`)
	bundle := regexp.MustCompile(`\b` + regexp.QuoteMeta(strings.ToLower(old)) + `\.bundle\b`)
	code, _ := filepath.Glob(filepath.Join(dir, srcNew, "*.cs"))
	for _, f := range code {
		_ = editFile(f, func(b string) string {
			return bundle.ReplaceAllString(word.ReplaceAllString(b, name), strings.ToLower(name)+".bundle")
		})
	}
	for _, f := range []string{"README.md", "DEVELOPING.md", filepath.Join("assets", "README.md")} {
		_ = editFile(filepath.Join(dir, f), func(b string) string {
			return bundle.ReplaceAllString(word.ReplaceAllString(b, name), strings.ToLower(name)+".bundle")
		})
	}
	if _, err := os.Stat(filepath.Join(dir, "assets", strings.ToLower(old)+".bundle")); err == nil {
		_ = move(filepath.Join("assets", strings.ToLower(old)+".bundle"), filepath.Join("assets", strings.ToLower(name)+".bundle"))
	}
	return filepath.Join(dir, srcNew, name+".csproj"), nil
}
