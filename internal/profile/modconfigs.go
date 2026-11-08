package profile

import (
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// ConfigAsideDir holds, by file name, the config files of mods switched off in the profile; switching the mod back on
// puts them back, so its settings survive the round trip. They are filed by base name alone, whatever config folder
// they came from, as 0.3.8 set them aside.
const ConfigAsideDir = "config-aside"

// bepInExConfigRoot is where BepInEx keeps mod settings, the only folder every release set aside and backed up: a
// profile an older release wrote keeps reading it whatever loader it now runs.
const bepInExConfigRoot = "BepInEx/config"

// loaderConfigRoots are the folders the profile's loader keeps mod settings in, by slash path below the profile's
// root: the loader's own, as a share carries them (share.LoaderConfigRoots), then BepInEx's, which an older release
// and a loader that names no folder leave as the only one.
func loaderConfigRoots(game string, p Profile) []string {
	var out []string
	if l, ok := gamereg.LoaderOf(game, p.LoaderFor(game)); ok {
		if c, ok := l.(loader.WithConfig); ok {
			for _, dir := range c.ConfigDirs() {
				if dir != "" && !slices.Contains(out, dir) {
					out = append(out, dir)
				}
			}
		}
	}
	if !slices.Contains(out, bepInExConfigRoot) {
		out = append(out, bepInExConfigRoot)
	}
	return out
}

// modConfigNames is what a mod's BepInEx config files are named after: its plugins' GUIDs and its package's name
// (LethalLevelLoader.cfg for thunderstore:IAmBatby-LethalLevelLoader).
func (s *Store) modConfigNames(game string, p Profile, id mod.ID) []string {
	names := s.pluginGUIDsOf(game, p, id)
	if _, name, ok := strings.Cut(id.Local(), "-"); ok && id.Format() != "" {
		names = append(names, name)
	}
	// BepInEx.cfg is the loader's own, whatever a mod is called.
	return slices.DeleteFunc(names, func(n string) bool { return strings.EqualFold(n, loaderConfigName) })
}

const loaderConfigName = "BepInEx"

// ownedConfigs is the .cfg files in folder named after one of names.
func ownedConfigs(folder string, names []string) []string {
	ents, found, err := safefs.ReadDirIfExists(folder)
	if err != nil || !found || len(names) == 0 {
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

// modConfigPaths is the live config files of the entry's mods that leave with the entry. It reads the profile as it
// is before the entry leaves.
func (s *Store) modConfigPaths(game string, p Profile, dir string, e Entry) []string {
	var out []string
	for _, root := range loaderConfigRoots(game, p) {
		folder := filepath.Join(dir, filepath.FromSlash(root))
		for _, m := range e.Mods {
			for _, f := range s.withoutEnabledOwned(game, p, e.Key, ownedConfigs(folder, s.modConfigNames(game, p, m.ID))) {
				out = append(out, filepath.Join(folder, f))
			}
		}
	}
	return out
}

// setConfigsAside moves the live config files of a removed entry into the profile's config-aside folder (the paths come
// from modConfigPaths), so Undo, or adding the mod again, brings its settings back through parkModConfigs.
func setConfigsAside(profileDir string, paths []string) error {
	aside := filepath.Join(profileDir, ConfigAsideDir)
	for _, f := range paths {
		to := filepath.Join(aside, filepath.Base(f))
		if err := datadir.MkdirParent(to); err != nil {
			return err
		}
		if err := safefs.Rename(f, to); err != nil {
			return err
		}
	}
	return nil
}

// parkModConfigs moves the config files of switched-off package mods aside and brings back those of mods switched on.
func (s *Store) parkModConfigs(game string, p Profile, dir string) error {
	aside := filepath.Join(dir, ConfigAsideDir)
	for _, root := range loaderConfigRoots(game, p) {
		live := filepath.Join(dir, filepath.FromSlash(root))
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
				if err := datadir.MkdirParent(filepath.Join(to, files[0])); err != nil {
					return err
				}
				if err := moveConfigs(files, from, to, aside); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// moveConfigs moves files from one config folder to the other. Back to live, a file already there is newer than the
// copy set aside (the mod wrote it since); going aside, the file already there is the older copy and is replaced.
func moveConfigs(files []string, from, to, aside string) error {
	for _, f := range files {
		dst := filepath.Join(to, f)
		if _, err := safefs.Stat(dst); err == nil {
			if to != aside {
				continue
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := safefs.Rename(filepath.Join(from, f), dst); err != nil {
			return err
		}
	}
	return nil
}
