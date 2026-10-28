package profile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// configAsideDir holds, by file name, the BepInEx config files of mods switched off in the profile; switching the mod
// back on puts them back, so its settings survive the round trip.
const configAsideDir = "config-aside"

// modConfigNames is what a mod's BepInEx config files are named after: its plugins' GUIDs and its package's name
// (LethalLevelLoader.cfg for thunderstore:IAmBatby-LethalLevelLoader).
func (s *Store) modConfigNames(game string, p Profile, id mod.ID) []string {
	names := s.pluginGUIDsOf(game, p, id)
	if _, name, ok := strings.Cut(id.Local(), "-"); ok && id.Format() != "" {
		names = append(names, name)
	}
	return names
}

// ownedConfigs is the .cfg files in folder named after one of names.
func ownedConfigs(folder string, names []string) []string {
	ents, err := os.ReadDir(folder)
	if err != nil || len(names) == 0 {
		return nil
	}
	var out []string
	for _, e := range ents {
		stem, ok := strings.CutSuffix(e.Name(), ".cfg")
		if !ok || e.IsDir() {
			continue
		}
		for _, n := range names {
			if strings.EqualFold(stem, n) {
				out = append(out, e.Name())
				break
			}
		}
	}
	return out
}

// withoutEnabledOwned drops from files, .cfg names in the profile's config folders, those another switched-on mod
// also owns, so a fork's config is not taken with the mod it forks. The entry with key skip is not counted.
func (s *Store) withoutEnabledOwned(game string, p Profile, skip string, files []string) []string {
	if len(files) == 0 {
		return nil
	}
	taken := map[string]bool{}
	for _, e := range p.Entries {
		if e.Key == skip {
			continue
		}
		for _, m := range e.Mods {
			if !e.Enabled(m.ID) {
				continue
			}
			for _, n := range s.modConfigNames(game, p, m.ID) {
				taken[strings.ToLower(n)] = true
			}
		}
	}
	return slices.DeleteFunc(slices.Clone(files), func(f string) bool {
		return taken[strings.ToLower(strings.TrimSuffix(f, ".cfg"))]
	})
}

// modConfigPaths is the config files of the entry's mods, live and set aside, that go when the entry is removed, so
// the mod starts fresh if it is added again. It reads the profile as it is before the entry leaves.
func (s *Store) modConfigPaths(game string, p Profile, dir string, e Entry) []string {
	var out []string
	for _, m := range e.Mods {
		names := s.modConfigNames(game, p, m.ID)
		for _, folder := range []string{filepath.Join(dir, "BepInEx", "config"), filepath.Join(dir, configAsideDir)} {
			for _, f := range s.withoutEnabledOwned(game, p, e.Key, ownedConfigs(folder, names)) {
				out = append(out, filepath.Join(folder, f))
			}
		}
	}
	return out
}

func removeConfigPaths(paths []string) {
	for _, f := range paths {
		_ = fsx.Remove(f)
	}
}

// parkModConfigs moves the config files of switched-off package mods aside and brings back those of mods switched on.
func (s *Store) parkModConfigs(game string, p Profile, dir string) error {
	live := filepath.Join(dir, "BepInEx", "config")
	aside := filepath.Join(dir, configAsideDir)
	for _, e := range p.Entries {
		if !e.Package {
			continue
		}
		for _, m := range e.Mods {
			from, to := aside, live
			if !e.Enabled(m.ID) {
				from, to = live, aside
			}
			files := ownedConfigs(from, s.modConfigNames(game, p, m.ID))
			if to == aside {
				files = s.withoutEnabledOwned(game, p, "", files)
			}
			if len(files) == 0 {
				continue
			}
			if err := os.MkdirAll(to, 0o750); err != nil {
				return err
			}
			for _, f := range files {
				dst := filepath.Join(to, f)
				// A file already where it would go is newer than the copy set aside: the mod wrote it since.
				if _, err := fsx.Stat(dst); err == nil {
					continue
				} else if !errors.Is(err, fs.ErrNotExist) {
					return err
				}
				if err := fsx.Rename(filepath.Join(from, f), dst); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
