package devsvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// loaderPackage is the loader itself, which every BepInEx mod depends on and none can drop.
const loaderPackage = "BepInEx-BepInExPack"

// Dependency is a package the project depends on; Soft when the mod runs without it.
type Dependency struct {
	Package string `json:"package"`
	Version string `json:"version"`
	Soft    bool   `json:"soft"`
}

var dependencyItem = regexp.MustCompile(`<(` + hardItem + `|` + softItem + `)\s+Include="([^"]+)"\s+Version="([^"]*)"`)

// Dependencies lists the packages the project in dir depends on, by name, the loader left out.
func (s *Service) Dependencies(gameID, dir string) ([]Dependency, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return nil, err
	}
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		return nil, usererr.Wrap(usererr.NotFound, errors.New("the project file is missing"))
	}
	raw, err := fsx.ReadFile(csproj)
	if err != nil {
		return nil, err
	}
	out := []Dependency{}
	for _, m := range dependencyItem.FindAllStringSubmatch(string(raw), -1) {
		if m[2] != loaderPackage {
			out = append(out, Dependency{Package: m[2], Version: m[3], Soft: m[1] == softItem})
		}
	}
	slices.SortFunc(out, func(a, b Dependency) int { return strings.Compare(a.Package, b.Package) })
	return out, nil
}

// RemoveDependency undoes AddDependency for pkg: its entry and DLL reference in the project file, its folder in the
// player's own props, its plugins' [BepInDependency] declarations and a soft dependency's compatibility class, which
// would no longer compile without it.
func (s *Service) RemoveDependency(gameID, dir, pkg string) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	if !thunderstorePackage.MatchString(pkg) || pkg == loaderPackage {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is not a dependency the project can drop", pkg))
	}
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		return usererr.Wrap(usererr.NotFound, errors.New("the project file is missing"))
	}
	raw, err := fsx.ReadFile(csproj)
	if err != nil {
		return err
	}
	props := filepath.Join(p.Dir, "Config.Build.user.props")
	userProps, _ := fsx.ReadFile(props)
	folder := readProp(string(userProps), depProp(pkg))
	soft := projectDependencies(string(raw))[pkg]
	return s.wire(context.Background(), p.Dir, func() error {
		if err := fsx.WriteFile(csproj, []byte(withoutDependency(string(raw), pkg)), 0o644); err != nil { // #nosec G306 -- a tracked project file
			return err
		}
		src := filepath.Dir(csproj)
		if folder != "" {
			for _, pl := range dotnet.PluginsIn(folder) {
				if err := undeclareDependency(src, pl.GUID); err != nil {
					return err
				}
			}
		}
		if soft {
			if err := dropUntouchedCompat(src, pkg); err != nil {
				return err
			}
		}
		if len(userProps) == 0 {
			return nil
		}
		return fsx.WriteFile(props, []byte(withoutUserDependency(string(userProps), pkg)), 0o600)
	})
}

// dropUntouchedCompat deletes a soft dependency's compatibility class only while it is exactly what writeCompat made,
// for any guid; a class the player has written into stays, theirs to rework.
func dropUntouchedCompat(src, pkg string) error {
	_, name, _ := strings.Cut(pkg, "-")
	class := notIdent.ReplaceAllString(name, "")
	if class == "" {
		return nil
	}
	path := filepath.Join(src, "Compat", class+"Compat.cs")
	raw, err := fsx.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	guid := regexp.MustCompile(`ContainsKey\("([^"]*)"\)`).FindStringSubmatch(string(raw))
	namespace := regexp.MustCompile(`(?m)^namespace ([^;]+)\.Compat;`).FindStringSubmatch(string(raw))
	if guid == nil || namespace == nil || string(raw) != fmt.Sprintf(compatTemplate, namespace[1], name, class, guid[1]) {
		return nil
	}
	return fsx.Remove(path)
}

// lineOf matches a whole line holding pattern, its line break included.
func lineOf(pattern string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^[ \t]*` + pattern + `[^\n]*\n?`)
}

func withoutDependency(project, pkg string) string {
	project = lineOf(`<(?:`+hardItem+`|`+softItem+`)\s+Include="`+regexp.QuoteMeta(pkg)+`"`).ReplaceAllString(project, "")
	return lineOf(`<Reference Include="\$\(`+regexp.QuoteMeta(depProp(pkg))+`\)`).ReplaceAllString(project, "")
}

func withoutUserDependency(props, pkg string) string {
	props = lineOf(`<`+regexp.QuoteMeta(depProp(pkg))+`>`).ReplaceAllString(props, "")
	return lineOf(`<MortarDependencyFolder Include="`+regexp.QuoteMeta(xmlEscape(pkg))+`"`).ReplaceAllString(props, "")
}

// undeclareDependency drops the plugin's [BepInDependency] for guid, hard or soft.
func undeclareDependency(src, guid string) error {
	path := pluginSource(src)
	if path == "" || !pluginGUID.MatchString(guid) {
		return nil
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	body := lineOf(`\[BepInDependency\("`+regexp.QuoteMeta(guid)+`"`).ReplaceAllString(string(raw), "")
	if body == string(raw) {
		return nil
	}
	return fsx.WriteFile(path, []byte(body), 0o644) // #nosec G306 -- a tracked source file
}
