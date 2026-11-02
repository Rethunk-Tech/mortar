package profile

import (
	"errors"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
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
	entries, err := s.store.entryFolders(game, dir, p)
	if err != nil {
		return nil, err
	}
	enabled := make([]string, len(entries))
	for i, e := range entries {
		enabled[i] = e.folder
	}
	var install string
	if s.settings != nil {
		install, _ = gamereg.InstallDir(s.home, s.settings.Get(), game)
	}
	rows, err := o.Order(loader.ProfileView{Game: game, Dir: dir, InstallDir: install, Enabled: enabled})
	if err != nil {
		return nil, err
	}
	for i, r := range rows {
		if j := slices.IndexFunc(entries, func(e entryFolder) bool { return r.Folder != "" && e.folder == r.Folder }); j >= 0 {
			rows[i].Entry = entries[j].key
		}
	}
	return rows, nil
}

// entryFolder is an enabled entry's key and folder.
type entryFolder struct{ key, folder string }

// enabledFolders are the folders of the profile's enabled user mods: a package's laid-out files in the store, and each
// other mod's own folder in mods/.
func (s *Store) enabledFolders(game, dir string, p Profile) ([]string, error) {
	entries, err := s.entryFolders(game, dir, p)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.folder
	}
	return out, nil
}

// entryFolders are enabledFolders with the key of the entry each folder comes from.
func (s *Store) entryFolders(game, dir string, p Profile) ([]entryFolder, error) {
	var out []entryFolder
	has := func(folder string) bool {
		return slices.ContainsFunc(out, func(e entryFolder) bool { return e.folder == folder })
	}
	modsDir := datadir.ModsDirOf(dir)
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		if e.Package {
			if s.items == nil || (len(e.Mods) > 0 && !slices.ContainsFunc(e.Mods, func(m Component) bool { return e.Enabled(m.ID) })) {
				continue
			}
			// A package whose files are no longer in the store has nothing to read.
			if itemDir, err := s.items.Path(game, e.StoreKey()); err == nil && !has(itemDir) {
				out = append(out, entryFolder{e.Key, itemDir})
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
			out = append(out, entryFolder{e.Key, plain})
		}
	}
	return out, nil
}

// EnabledFolders are the folders of the profile's enabled mods in the profile's order.
func (s *Store) EnabledFolders(game, id string) ([]string, error) {
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	return s.enabledFolders(game, dir, p)
}
