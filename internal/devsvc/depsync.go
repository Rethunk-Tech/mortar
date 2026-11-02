package devsvc

import (
	"slices"
	"strings"

	tstore "github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// DepPackage is a Thunderstore package of the development profile at its installed version.
type DepPackage struct {
	Package string `json:"package"`
	Version string `json:"version"`
}

// DepVersion is a declared dependency whose installed version differs from the declared one.
type DepVersion struct {
	Package   string `json:"package"`
	Declared  string `json:"declared"`
	Installed string `json:"installed"`
	Soft      bool   `json:"soft"`
}

// DepSync is how a project's declared Thunderstore dependencies and its development profile's mods differ.
type DepSync struct {
	// Profile is the project's development profile, where missing dependencies install whether or not it is open.
	Profile string `json:"profile"`
	// Missing are required dependencies the profile does not hold.
	Missing []Dependency `json:"missing"`
	// Undeclared are enabled mods the player added that the project does not declare.
	Undeclared []DepPackage `json:"undeclared"`
	Differs    []DepVersion `json:"differs"`
}

// diffDependencies compares the declared dependencies with the profile's Thunderstore mods. A soft dependency is not
// missing, since the mod runs without it. Mods the profile brought in for another mod (implicit), the loader, the
// project's own package and the ignored ones are not offered as undeclared.
func diffDependencies(declared []Dependency, installed []profile.Installed, ignored, own []string, shared bool) DepSync {
	out := DepSync{Missing: []Dependency{}, Undeclared: []DepPackage{}, Differs: []DepVersion{}}
	held := map[string]profile.Installed{}
	// A package in the profile from a local folder, such as another project's own build: it is there, and offering the
	// published version would put a second copy beside it.
	local := map[string]bool{}
	for _, im := range installed {
		switch {
		case im.Source.Kind == profile.KindThunderstore:
			held[strings.ToLower(im.Source.Name)] = im
		case mod.ID(im.UniqueID).Format() == mod.FormatThunderstore:
			local[strings.ToLower(mod.ID(im.UniqueID).Local())] = true
		}
	}
	isDeclared := map[string]bool{}
	for _, d := range declared {
		isDeclared[strings.ToLower(d.Package)] = true
		im, ok := held[strings.ToLower(d.Package)]
		switch {
		case local[strings.ToLower(d.Package)]:
		case !ok:
			if !d.Soft {
				out.Missing = append(out.Missing, d)
			}
		case d.Version != "" && im.Source.Version != "" && versionOf(d.Package, d.Version) != versionOf(d.Package, im.Source.Version):
			out.Differs = append(out.Differs, DepVersion{Package: d.Package, Declared: d.Version, Installed: im.Source.Version, Soft: d.Soft})
		}
	}
	// In a profile several projects are developed in, a mod may be there for any of them, or just for playing.
	if shared {
		return out
	}
	seen := map[string]bool{}
	for _, im := range installed {
		key := strings.ToLower(im.Source.Name)
		if im.Source.Kind != profile.KindThunderstore || !im.Enabled || im.Implicit || isDeclared[key] || seen[key] ||
			strings.EqualFold(im.Source.Name, loaderPackage) ||
			slices.ContainsFunc(ignored, func(s string) bool { return strings.EqualFold(s, im.Source.Name) }) ||
			slices.ContainsFunc(own, func(s string) bool { return strings.EqualFold(s, im.Source.Name) }) {
			continue
		}
		seen[key] = true
		out.Undeclared = append(out.Undeclared, DepPackage{Package: im.Source.Name, Version: im.Source.Version})
	}
	return out
}

// versionOf is the version of a Team-Name package written as Team-Name-version, or the text itself when that does not
// parse.
func versionOf(pkg, version string) string {
	d, err := tstore.ParseDependency(pkg + "-" + version)
	if err != nil {
		return version
	}
	return d.Version
}

// DependencySync compares the project's declared dependencies with the mods of its development profile. It changes
// nothing; the player picks what to install, declare or update.
func (s *Service) DependencySync(gameID, dir string) (DepSync, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return DepSync{}, err
	}
	declared, err := s.Dependencies(gameID, dir)
	if err != nil {
		return DepSync{}, err
	}
	installed, err := s.Profiles.Installed(gameID, p.Profile)
	if err != nil {
		return DepSync{}, err
	}
	var own []string
	sharing := 0
	for _, q := range s.Projects(gameID) {
		if q.Profile == p.Profile {
			sharing++
		}
		if pkg := packageOf(q.Dir); pkg != "" {
			own = append(own, pkg)
		}
	}
	out := diffDependencies(declared, installed, p.IgnoredDeps, own, sharing > 1)
	out.Profile = p.Profile
	return out, nil
}

// IgnoreDependency stops offering pkg as a dependency of the project in dir.
func (s *Service) IgnoreDependency(gameID, dir, pkg string) error {
	if !thunderstorePackage.MatchString(pkg) {
		return usererr.New(usererr.Invalid, pkg+" is not a Thunderstore package")
	}
	if _, err := s.project(gameID, dir); err != nil {
		return err
	}
	return s.Settings.IgnoreDevProjectDependency(gameID, dir, pkg)
}
