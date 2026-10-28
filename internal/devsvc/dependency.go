package devsvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/msbuild"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// thunderstorePackage is a Thunderstore package's Team-Name, the form manifest.json's dependencies use.
var thunderstorePackage = regexp.MustCompile(`^[A-Za-z0-9_]+-[A-Za-z0-9_]+$`)

const (
	// hardItem is what the template's pack step writes into manifest.json; softItem only records the package.
	hardItem = "ThunderstoreDependency"
	softItem = "ThunderstoreSoftDependency"
	// depPropPrefix starts the Config.Build.user.props property holding a dependency's folder in Mortar's store, which
	// its DLL reference is relative to; the folder moves with each update, so every build refreshes it.
	depPropPrefix = "MortarDependency_"
)

// AddDependency makes pkg at version a dependency of the project in dir and references its DLLs from the development
// profile without copying them into the build. A hard dependency joins manifest.json; a soft one only the project, and
// the mod runs without it. Each is declared on the plugin once its DLLs are installed (syncDependencies).
func (s *Service) AddDependency(gameID, dir, pkg, version string, soft bool) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	if !thunderstorePackage.MatchString(pkg) || !packageVersion.MatchString(version) {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s %s is not a Thunderstore package and version", pkg, version))
	}
	if p.Kind != "" {
		// A soft dependency means nothing to a package of files; the loader it needs is required.
		return s.wire(context.Background(), p.Dir, p.Name, func() error { return setManifestDependency(p.Dir, pkg, version) })
	}
	csproj, raw, err := readCsproj(p.Dir)
	if err != nil {
		return err
	}
	return s.wire(context.Background(), p.Dir, p.Name, func() error {
		if err := fsx.WriteFile(csproj, []byte(withDependency(string(raw), pkg, version, soft)), 0o644); err != nil { // #nosec G306 -- a tracked project file
			return err
		}
		return s.syncDependencies(gameID, p)
	})
}

// wire runs fn, which writes a project's dependency wiring, and while Mortar's own "Start" commit is still the latest
// one and on no remote, folds the files fn changed into it: New mod adds its dependencies as they install, after that
// commit, and a new project should start with nothing to commit. Files changed before fn ran are left alone.
func (s *Service) wire(ctx context.Context, dir, name string, fn func() error) error {
	before, ok := s.dirty(ctx, dir)
	if err := fn(); err != nil {
		return err
	}
	if ok && s.startIsLatest(ctx, dir, name) {
		s.foldIntoStart(ctx, dir, before)
	}
	return nil
}

// startIsLatest reports whether the project's latest commit is Mortar's "Start" commit and no remote has it. That
// commit is the first one, or the second after the initial commit GitHub makes when it copies the template.
func (s *Service) startIsLatest(ctx context.Context, dir, name string) bool {
	subject, err := s.run(ctx, dir, "git", "log", "-1", "--format=%s")
	if err != nil || strings.TrimSpace(string(subject)) != "Start "+name {
		return false
	}
	if count, err := s.run(ctx, dir, "git", "rev-list", "--count", "HEAD"); err != nil || !slices.Contains([]string{"1", "2"}, strings.TrimSpace(string(count))) {
		return false
	}
	remote, err := s.run(ctx, dir, "git", "branch", "-r", "--contains", "HEAD")
	return err == nil && strings.TrimSpace(string(remote)) == ""
}

// foldIntoStart amends the latest commit with the files changed since before was listed.
func (s *Service) foldIntoStart(ctx context.Context, dir string, before map[string]bool) {
	after, _ := s.dirty(ctx, dir)
	var paths []string
	for path := range after {
		if !before[path] {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return
	}
	if _, err := s.run(ctx, dir, "git", append([]string{"add", "--"}, paths...)...); err != nil {
		log.Printf("devsvc: staging %s's dependencies: %v", dir, err)
		return
	}
	if _, err := s.run(ctx, dir, "git", append([]string{"commit", "-q", "--amend", "--no-edit", "--"}, paths...)...); err != nil {
		log.Printf("devsvc: folding dependencies into %s's first commit: %v", dir, err)
	}
}

// dirty lists the project's changed and untracked files by path; ok is false outside a git repository.
func (s *Service) dirty(ctx context.Context, dir string) (map[string]bool, bool) {
	out, err := s.run(ctx, dir, "git", "status", "--porcelain", "-uall", "-z")
	if err != nil {
		return nil, false
	}
	paths := map[string]bool{}
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry) <= 3 {
			continue
		}
		paths[entry[3:]] = true
		// A rename or copy is followed by its original path as a field of its own, without a status prefix.
		if strings.ContainsAny(entry[:2], "RC") {
			i++
		}
	}
	return paths, true
}

// withDependency sets pkg as a hard or soft dependency in a project file, in place of whichever kind it was or after
// the last dependency, and adds its DLL reference once.
func withDependency(project, pkg, version string, soft bool) string {
	kind := hardItem
	if soft {
		kind = softItem
	}
	item := msbuild.Item{Type: kind, Include: pkg, Attrs: map[string]string{"Version": version}}
	out, err := msbuild.SetItem(project, item, []string{hardItem, softItem}, isDependencyItem)
	if err != nil {
		return project
	}
	return withReference(out, pkg)
}

// withReference adds pkg's DLL reference, relative to its folder property, after the last dependency unless it is there.
func withReference(project, pkg string) string {
	ref := depReference(pkg)
	out, err := msbuild.SetItem(project, ref, []string{"Reference"}, isDependencyItem)
	if err != nil {
		return project
	}
	return out
}

func depReference(pkg string) msbuild.Item {
	prop := depProp(pkg)
	return msbuild.Item{Type: "Reference", Include: "$(" + prop + ")**/*.dll", Attrs: map[string]string{"Condition": "'$(" + prop + ")' != ''", "Private": "false"}}
}

// isDependencyItem picks a dependency or the reference to one, the items a new one goes after.
func isDependencyItem(it msbuild.Item) bool {
	return it.Type == hardItem || it.Type == softItem || it.Type == "Reference" && strings.HasPrefix(it.Include, "$("+depPropPrefix)
}

// depProp names the user property holding pkg's folder.
func depProp(pkg string) string { return depPropPrefix + strings.ReplaceAll(pkg, "-", "_") }

// editUserProps changes the player's own Config.Build.user.props, kept out of the repository, starting one when there is none.
func editUserProps(dir string, fn func(string) (string, error)) error {
	path := filepath.Join(dir, "Config.Build.user.props")
	raw, err := fsx.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	body := string(raw)
	if strings.TrimSpace(body) == "" {
		body = "<Project>\n</Project>\n"
	}
	out, err := fn(body)
	if err != nil || out == string(raw) {
		return err
	}
	return fsx.WriteFile(path, []byte(out), 0o600)
}

// setUserDependency points pkg's folder property at folder and lists pkg as a MortarDependencyFolder, so the template
// leaves fetching it to Mortar.
func setUserDependency(dir, pkg, folder string) error {
	return editUserProps(dir, func(body string) (string, error) {
		body, err := msbuild.SetProperty(body, depProp(pkg), folder)
		if err != nil {
			return body, err
		}
		item := msbuild.Item{Type: "MortarDependencyFolder", Include: pkg}
		return msbuild.SetItem(body, item, []string{item.Type}, func(it msbuild.Item) bool { return it.Type == item.Type })
	})
}

// packageOf is the Thunderstore package (Team-Name) the project in dir publishes, empty when it names no team.
func packageOf(dir string) string {
	csproj, ok := csprojOf(dir)
	if !ok {
		return ""
	}
	project, _ := fsx.ReadFile(csproj)
	props, _ := fsx.ReadFile(filepath.Join(dir, propsFile))
	team := msbuild.Property(string(props), "ThunderstoreTeam")
	if team == "" {
		return ""
	}
	return team + "-" + cmpOr(msbuild.Property(string(project), "ThunderstoreName"), strings.TrimSuffix(filepath.Base(csproj), ".csproj"))
}

// projectDependencies reads a project file's dependencies by package: true for a soft one.
func projectDependencies(project string) map[string]bool {
	out := map[string]bool{}
	items, _ := msbuild.Items(project, hardItem, softItem)
	for _, it := range items {
		out[it.Include] = it.Type == softItem
	}
	return out
}

// syncDependencies points each dependency's reference at its folder in the development profile, declares its plugins
// on the mod's plugin class, hard or soft, and gives a soft one its compatibility class. A dependency not installed
// there yet waits for the next build.
func (s *Service) syncDependencies(gameID string, p settings.DevProject) error {
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		return nil
	}
	raw, err := fsx.ReadFile(csproj)
	if err != nil {
		return err
	}
	deps := projectDependencies(string(raw))
	if len(deps) == 0 {
		return nil
	}
	installed, err := s.Profiles.Installed(gameID, p.Profile)
	if err != nil {
		return err
	}
	folders := map[string]string{}
	for _, im := range installed {
		if im.Source.Kind == profile.KindThunderstore && im.Folder != "" {
			folders[im.Source.Name] = im.Folder
		}
	}
	// Another project of the game that publishes the package is built from here, and may not be on Thunderstore yet.
	for _, q := range s.Projects(gameID) {
		if q.Dir != p.Dir && q.Output != "" {
			if pkg := packageOf(q.Dir); pkg != "" {
				folders[pkg] = q.Output
			}
		}
	}
	src := filepath.Dir(csproj)
	namespace := cmpOr(msbuild.Property(string(raw), "RootNamespace"), strings.TrimSuffix(filepath.Base(csproj), ".csproj"))
	// A dependency the template declared has no reference of its own, since the build downloaded it; once Mortar points
	// at a folder for it, the build skips that download and needs the reference.
	referenced := string(raw)
	for pkg := range deps {
		if folders[pkg] != "" && pkg != "BepInEx-BepInExPack" {
			referenced = withReference(referenced, pkg)
		}
	}
	if referenced != string(raw) {
		if err := fsx.WriteFile(csproj, []byte(referenced), 0o644); err != nil { // #nosec G306 -- a tracked project file
			return err
		}
	}
	for pkg, soft := range deps {
		folder := folders[pkg]
		if folder == "" {
			continue
		}
		if err := setUserDependency(p.Dir, pkg, folder+string(filepath.Separator)); err != nil {
			return err
		}
		for _, pl := range dotnet.PluginsIn(folder) {
			if err := declareDependency(src, pl.GUID, soft); err != nil {
				return err
			}
			if soft {
				if err := writeCompat(src, namespace, pkg, pl.GUID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// readCsproj reads the project file of the project in dir.
func readCsproj(dir string) (path string, raw []byte, err error) {
	path, ok := csprojOf(dir)
	if !ok {
		return "", nil, usererr.Wrap(usererr.NotFound, errors.New("the project file is missing"))
	}
	raw, err = fsx.ReadFile(path)
	return path, raw, err
}
