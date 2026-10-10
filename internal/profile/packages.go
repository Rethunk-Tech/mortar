package profile

import (
	"cmp"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// PackageRef is one package of a profile (a BepInEx-style install from any source) and the store folder holding its
// files.
type PackageRef struct {
	Key string
	// Enabled is false for a package every component of which is switched off.
	Enabled bool
	// Source is the entry's source kind; only a Thunderstore package's Name is a "Namespace-Name".
	Source string
	// ID is the id of the package component, which enabling or disabling the package takes.
	ID      mod.ID
	Name    string
	Version string
	Dir     string
}

// EnabledPackages lists the profile's enabled packages, whatever their source.
func (s *Store) EnabledPackages(game, id string) ([]PackageRef, error) {
	all, err := s.Packages(game, id)
	return slices.DeleteFunc(all, func(p PackageRef) bool { return !p.Enabled }), err
}

// Packages lists the profile's packages, enabled or not, whatever their source. A package whose files are no longer in
// the store is left out, since nothing can be read from it.
func (s *Store) Packages(game, id string) ([]PackageRef, error) {
	p, err := s.read(game, id)
	if err != nil {
		return nil, err
	}
	var out []PackageRef
	for _, e := range p.Entries {
		if !e.Package || s.items == nil {
			continue
		}
		enabled := len(e.Mods) == 0 || slices.ContainsFunc(e.Mods, func(m Component) bool { return e.Enabled(m.ID) })
		dir, err := s.items.Path(game, e.StoreKey())
		if err != nil {
			continue
		}
		var id mod.ID
		name, version := e.Source.Name, e.Source.Version
		if len(e.Mods) > 0 {
			id = e.Mods[0].ID
			if e.Source.Kind != KindThunderstore {
				name = e.Mods[0].Name
			}
			// A package installed from a file has no source version; its manifest's is the one it has.
			version = cmp.Or(version, e.Mods[0].Version)
		}
		out = append(out, PackageRef{Key: e.Key, Enabled: enabled, Source: e.Source.Kind, ID: id, Name: name, Version: version, Dir: dir})
	}
	return out, nil
}
