package profile

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// PackageRef is one enabled package of a profile (a BepInEx-style install from any source) and the store folder
// holding its files.
type PackageRef struct {
	Key string
	// Source is the entry's source kind; only a Thunderstore package's Name is a "Namespace-Name".
	Source string
	// ID is the id of the package component, which enabling or disabling the package takes.
	ID      mod.ID
	Name    string
	Version string
	Dir     string
}

// EnabledPackages lists the profile's enabled packages, whatever their source. A package whose files are no longer in the
// store is left out, since nothing can be read from it.
func (s *Store) EnabledPackages(game, id string) ([]PackageRef, error) {
	p, err := s.read(game, id)
	if err != nil {
		return nil, err
	}
	var out []PackageRef
	for _, e := range p.Entries {
		if !e.Package || s.items == nil {
			continue
		}
		if len(e.Mods) > 0 && !slices.ContainsFunc(e.Mods, func(m Component) bool { return e.Enabled(m.ID) }) {
			continue
		}
		dir, err := s.items.Path(game, e.Key)
		if err != nil {
			continue
		}
		var id mod.ID
		name := e.Source.Name
		if len(e.Mods) > 0 {
			id = e.Mods[0].ID
			if e.Source.Kind != KindThunderstore {
				name = e.Mods[0].Name
			}
		}
		out = append(out, PackageRef{Key: e.Key, Source: e.Source.Kind, ID: id, Name: name, Version: e.Source.Version, Dir: dir})
	}
	return out, nil
}
