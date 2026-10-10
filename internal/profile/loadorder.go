package profile

import (
	"errors"
	"path/filepath"
	"slices"

	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
)

// LoadOrder lists the profile's enabled mods in the order its loader loads them.
func (s *Service) LoadOrder(game, id string) ([]loadorder.Row, error) {
	l, _ := gamereg.LoaderOf(game, s.store.LoaderID(game, id))
	o, ok := l.(loader.WithOrder)
	if !ok {
		return nil, errors.New("the profile's loader has no load order")
	}
	p, dir, err := s.store.readDir(game, id)
	if err != nil {
		return nil, err
	}
	enabled, err := s.store.enabledFolders(game, dir, p)
	if err != nil {
		return nil, err
	}
	return o.Order(loader.ProfileView{Game: game, Dir: dir, Enabled: enabled})
}

// enabledFolders are the folders of the profile's enabled user mods: a package's laid-out files in the store, and each
// other mod's own folder in mods/.
func (s *Store) enabledFolders(game, dir string, p Profile) ([]string, error) {
	var out []string
	modsDir := filepath.Join(dir, "mods")
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		if e.Package {
			if s.items == nil || (len(e.Mods) > 0 && !slices.ContainsFunc(e.Mods, func(m Component) bool { return e.Enabled(m.ID) })) {
				continue
			}
			// A package whose files are no longer in the store has nothing to read.
			if itemDir, err := s.items.Path(game, e.StoreKey()); err == nil && !slices.Contains(out, itemDir) {
				out = append(out, itemDir)
			}
			continue
		}
		for _, m := range e.Mods {
			if !e.Enabled(m.ID) {
				continue
			}
			plain, _, err := ModPaths(modsDir, e.Key, m.Folder)
			if err != nil {
				return nil, err
			}
			out = append(out, plain)
		}
	}
	return out, nil
}
