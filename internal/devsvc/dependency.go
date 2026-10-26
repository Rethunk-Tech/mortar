package devsvc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		return usererr.Wrap(usererr.NotFound, errors.New("the project file is missing"))
	}
	raw, err := fsx.ReadFile(csproj)
	if err != nil {
		return err
	}
	if err := fsx.WriteFile(csproj, []byte(withDependency(string(raw), pkg, version, soft)), 0o644); err != nil { // #nosec G306 -- a tracked project file
		return err
	}
	return s.syncDependencies(gameID, p)
}

// withDependency sets pkg as a hard or soft dependency in a project file, replacing whichever kind it was, and adds
// its DLL reference once: in place of an existing entry, after the last dependency, or in a new ItemGroup at the end.
func withDependency(project, pkg, version string, soft bool) string {
	kind := hardItem
	if soft {
		kind = softItem
	}
	item := fmt.Sprintf(`<%s Include="%s" Version="%s" />`, kind, pkg, version)
	listed := regexp.MustCompile(`<(?:` + hardItem + `|` + softItem + `)\s+Include="` + regexp.QuoteMeta(pkg) + `"[^>]*/>`)
	if listed.MatchString(project) {
		project = listed.ReplaceAllLiteralString(project, item)
	} else {
		project = afterLastDependency(project, item)
	}
	prop := depProp(pkg)
	ref := fmt.Sprintf(`<Reference Include="$(%[1]s)**/*.dll" Condition="'$(%[1]s)' != ''" Private="false" />`, prop)
	if !strings.Contains(project, ref) {
		project = afterLastDependency(project, ref)
	}
	return project
}

var dependencyLine = regexp.MustCompile(`(?m)^([ \t]*)<(?:` + hardItem + `|` + softItem + `|Reference Include="\$\(` + depPropPrefix + `)[^\n]*/>`)

func afterLastDependency(project, item string) string {
	if all := dependencyLine.FindAllStringSubmatchIndex(project, -1); len(all) > 0 {
		last := all[len(all)-1]
		indent := project[last[2]:last[3]]
		return project[:last[1]] + "\n" + indent + item + project[last[1]:]
	}
	closing := strings.LastIndex(project, "</Project>")
	if closing < 0 {
		return project
	}
	return project[:closing] + "  <ItemGroup>\n    " + item + "\n  </ItemGroup>\n" + project[closing:]
}

// depProp names the user property holding pkg's folder.
func depProp(pkg string) string { return depPropPrefix + strings.ReplaceAll(pkg, "-", "_") }

// setUserProp sets a property in the player's own Config.Build.user.props, kept out of the repository.
func setUserProp(dir, name, value string) error {
	path := filepath.Join(dir, "Config.Build.user.props")
	raw, err := fsx.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	body := string(raw)
	element := "<" + name + ">" + xmlEscape(value) + "</" + name + ">"
	switch {
	case readProp(body, name) != "":
		body = setProp(body, name, value)
	case strings.Contains(body, "</PropertyGroup>"):
		body = strings.Replace(body, "</PropertyGroup>", "  "+element+"\n  </PropertyGroup>", 1)
	default:
		body = "<Project>\n  <PropertyGroup>\n    " + element + "\n  </PropertyGroup>\n</Project>\n"
	}
	if body == string(raw) {
		return nil
	}
	return fsx.WriteFile(path, []byte(body), 0o600)
}

// setUserDependency points pkg's folder property at folder and lists pkg as a MortarDependencyFolder, so the template
// leaves fetching it to Mortar.
func setUserDependency(dir, pkg, folder string) error {
	if err := setUserProp(dir, depProp(pkg), folder); err != nil {
		return err
	}
	path := filepath.Join(dir, "Config.Build.user.props")
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	body := string(raw)
	item := `<MortarDependencyFolder Include="` + xmlEscape(pkg) + `" />`
	switch {
	case strings.Contains(body, item):
		return nil
	case strings.Contains(body, "</ItemGroup>"):
		body = strings.Replace(body, "</ItemGroup>", "  "+item+"\n  </ItemGroup>", 1)
	default:
		body = strings.Replace(body, "</Project>", "  <ItemGroup>\n    "+item+"\n  </ItemGroup>\n</Project>", 1)
	}
	return fsx.WriteFile(path, []byte(body), 0o600)
}

// projectDependencies reads a project file's dependencies by package: true for a soft one.
func projectDependencies(project string) map[string]bool {
	out := map[string]bool{}
	for _, m := range regexp.MustCompile(`<(`+hardItem+`|`+softItem+`)\s+Include="([^"]+)"`).FindAllStringSubmatch(project, -1) {
		out[m[2]] = m[1] == softItem
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
	src := filepath.Dir(csproj)
	namespace := cmpOr(readProp(string(raw), "RootNamespace"), strings.TrimSuffix(filepath.Base(csproj), ".csproj"))
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
