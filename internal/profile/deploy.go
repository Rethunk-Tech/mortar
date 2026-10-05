package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// placedFile lists, in the profile's root, the files SyncPackages put there, so it can take them away again.
const placedFile = ".mortar-packages.json"

// packageFiles are the files the profile's enabled packages lay out below the profile's root, by slash path, each the
// package that wins it (lowest priority first, so a later entry overrides), and how many files each entry wins over an
// earlier one.
func (s *Store) packageFiles(gameID, id string) (files map[string]packageFile, wins map[string]int, err error) {
	info, ok := components.BundledGame(gameID)
	if !ok || info.Deploy != components.DeployProfile {
		return nil, map[string]int{}, nil
	}
	p, _, err := s.readDir(gameID, id)
	if err != nil {
		return nil, nil, err
	}
	files, wins = map[string]packageFile{}, map[string]int{}
	for _, e := range p.Entries {
		if e.IsOverlay() || !e.hasPackageEnabled() {
			continue
		}
		arch, l, _, err := s.layoutOf(gameID, id, e.Key, e.Fomod)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", entryLabel(e), err)
		}
		for _, f := range l.Files {
			rel := path.Clean(f.Rel)
			if !filepath.IsLocal(filepath.FromSlash(rel)) {
				return nil, nil, fmt.Errorf("%s: %s leaves the profile", entryLabel(e), f.Rel)
			}
			if _, taken := files[rel]; taken {
				wins[e.Key]++
			}
			files[rel] = packageFile{src: filepath.Join(arch.Dir, filepath.FromSlash(f.Src)), key: e.Key}
		}
	}
	return files, wins, nil
}

type packageFile struct{ src, key string }

// PackageOverrides counts, for each entry key, the files it wins over an earlier package. A game that is
// redirected holds its mods in the profile's mods folder and has none.
func (s *Store) PackageOverrides(gameID, id string) (map[string]int, error) {
	_, wins, err := s.packageFiles(gameID, id)
	return wins, err
}

// SyncPackages lays the enabled packages' files out in the profile, where the loader reads them, and takes away the
// files of packages no longer enabled. A package's config file only seeds the profile: once it is there the player's
// edits stay. A game that is redirected has nothing to sync.
func (s *Store) SyncPackages(gameID, id string) error {
	files, _, err := s.packageFiles(gameID, id)
	if err != nil || files == nil {
		return err
	}
	_, dir, err := s.readDir(gameID, id)
	if err != nil {
		return err
	}
	var prev []string
	if b, err := fsx.ReadFile(filepath.Join(dir, placedFile)); err == nil {
		_ = json.Unmarshal(b, &prev)
	}
	for _, rel := range prev {
		if _, keep := files[rel]; !keep {
			removeUp(dir, filepath.Join(dir, filepath.FromSlash(rel)))
		}
	}
	var placed []string
	for rel := range files {
		if !strings.HasPrefix(rel, "BepInEx/config/") {
			placed = append(placed, rel)
		}
	}
	slices.Sort(placed)
	// The record goes first and names the old files too: a sync that stops part way still names every file it or an
	// earlier sync may have put there, so the next one can take them away.
	if err := writePlaced(dir, append(slices.Clone(prev), placed...)); err != nil {
		return err
	}
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if strings.HasPrefix(rel, "BepInEx/config/") {
			if _, err := os.Lstat(dst); err == nil {
				continue
			}
		} else if placedCurrent(files[rel].src, dst) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		if err := os.Remove(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := datadir.CopyFile(files[rel].src, dst); err != nil {
			return err
		}
	}
	return writePlaced(dir, placed)
}

func writePlaced(dir string, rels []string) error {
	slices.Sort(rels)
	b, err := json.Marshal(slices.Compact(rels))
	if err != nil {
		return err
	}
	return fsx.WriteFile(filepath.Join(dir, placedFile), b, 0o600)
}

// placedCurrent reports a placed file that still matches its source: same size and not older.
func placedCurrent(src, dst string) bool {
	a, err := os.Stat(src)
	if err != nil {
		return false
	}
	b, err := os.Stat(dst)
	return err == nil && a.Size() == b.Size() && !b.ModTime().Before(a.ModTime())
}

// removeUp removes a file, then the folders it leaves empty below root.
func removeUp(root, file string) {
	if os.Remove(file) != nil {
		return
	}
	for d := filepath.Dir(file); d != root && datadir.UnderRoot(root, d) && os.Remove(d) == nil; d = filepath.Dir(d) {
	}
}

// hasPackageEnabled reports an entry with no mods to switch off, or with at least one switched on.
func (e Entry) hasPackageEnabled() bool {
	return len(e.Mods) == 0 || slices.ContainsFunc(e.Mods, func(m Component) bool { return e.Enabled(m.ID) })
}
