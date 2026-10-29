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

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/msbuild"
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

// Dependencies lists the packages the project in dir depends on, by name, the loader left out.
func (s *Service) Dependencies(gameID, dir string) ([]Dependency, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return nil, err
	}
	if p.Kind != "" {
		return manifestDependencies(p.Dir)
	}
	_, raw, err := readCsproj(p.Dir)
	if err != nil {
		return nil, err
	}
	items, err := msbuild.Items(string(raw), hardItem, softItem)
	if err != nil {
		return nil, err
	}
	out := []Dependency{}
	for _, it := range items {
		if it.Include != loaderPackage {
			out = append(out, Dependency{Package: it.Include, Version: it.Attrs["Version"], Soft: it.Type == softItem})
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
	if p.Kind != "" {
		return s.wire(context.Background(), p, func() error { return setManifestDependency(p.Dir, pkg, "") })
	}
	csproj, raw, err := readCsproj(p.Dir)
	if err != nil {
		return err
	}
	props := filepath.Join(p.Dir, "Config.Build.user.props")
	userProps, _ := safefs.ReadFile(props)
	folder := msbuild.Property(string(userProps), depProp(pkg))
	soft := projectDependencies(string(raw))[pkg]
	return s.wire(context.Background(), p, func() error {
		project, err := withoutDependency(string(raw), pkg)
		if err != nil {
			return err
		}
		if err := safefs.WriteFile(csproj, []byte(project), 0o644); err != nil { // #nosec G306 -- a tracked project file
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
		return editUserProps(p.Dir, func(body string) (string, error) { return withoutUserDependency(body, pkg) })
	})
}

// dropUntouchedCompat deletes a soft dependency's compatibility class only while it is exactly what writeCompat made,
// for any guid; a class the player has written into stays, theirs to rework.
func dropUntouchedCompat(src, pkg string) error {
	name, class, path := compatFile(src, pkg)
	if class == "" {
		return nil
	}
	raw, err := safefs.ReadFile(path)
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
	return safefs.Remove(path)
}

// lineOf matches a whole line holding pattern, its line break included.
func lineOf(pattern string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^[ \t]*` + pattern + `[^\n]*\n?`)
}

func withoutDependency(project, pkg string) (string, error) {
	ref := depReference(pkg).Include
	return msbuild.Remove(project, func(e msbuild.Elem, parent string) bool {
		return parent == "ItemGroup" && (strings.EqualFold(e.Name, hardItem) || strings.EqualFold(e.Name, softItem)) && strings.EqualFold(e.Attrs["Include"], pkg) ||
			parent == "ItemGroup" && strings.EqualFold(e.Name, "Reference") && strings.EqualFold(e.Attrs["Include"], ref)
	})
}

func withoutUserDependency(props, pkg string) (string, error) {
	props, err := msbuild.RemoveProperty(props, depProp(pkg))
	if err != nil {
		return props, err
	}
	return msbuild.Remove(props, func(e msbuild.Elem, parent string) bool {
		return parent == "ItemGroup" && strings.EqualFold(e.Name, "MortarDependencyFolder") && strings.EqualFold(e.Attrs["Include"], pkg)
	})
}

// undeclareDependency drops the plugin's [BepInDependency] for guid, hard or soft.
func undeclareDependency(src, guid string) error {
	path := pluginSource(src)
	if path == "" || !pluginGUID.MatchString(guid) {
		return nil
	}
	raw, err := safefs.ReadFile(path)
	if err != nil {
		return err
	}
	body := lineOf(`\[BepInDependency\("`+regexp.QuoteMeta(guid)+`"`).ReplaceAllString(string(raw), "")
	if body == string(raw) {
		return nil
	}
	return safefs.WriteFile(path, []byte(body), 0o644) // #nosec G306 -- a tracked source file
}
