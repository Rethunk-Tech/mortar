package profile

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

// HiddenMod is a mod inside an entry whose folder, or a folder above it, starts with a dot that Mortar did not add,
// so SMAPI skips it and the entry does not list it.
type HiddenMod struct {
	Key string `json:"key"`
	// Folder is the mod's folder relative to the entry folder, with its dots, slash-separated.
	Folder   string `json:"folder"`
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	Author   string `json:"author"`
	Version  string `json:"version"`
}

// DotHiddenMods lists the dot-hidden mods in the profile's user entries.
func (s *Store) DotHiddenMods(game, id string) ([]HiddenMod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.read(game, id)
	if err != nil {
		return nil, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return nil, err
	}
	modsDir := filepath.Join(dir, "mods")
	out := []HiddenMod{}
	for _, e := range p.Entries {
		if isBundled(e) {
			continue
		}
		root := filepath.Join(modsDir, e.Key)
		if !exists(root) {
			root = filepath.Join(modsDir, "."+e.Key)
		}
		// Folders Mortar dotted to switch a mod off are the entry's own, not hidden.
		ours := map[string]bool{}
		for _, m := range e.Mods {
			if m.Folder != "." {
				ours[path.Join(path.Dir(m.Folder), "."+path.Base(m.Folder))] = true
			}
		}
		found, err := hiddenUnder(root, ours)
		if err != nil {
			return nil, err
		}
		for _, h := range found {
			h.Key = e.Key
			out = append(out, h)
		}
	}
	return out, nil
}

// DotHiddenMods lists mods inside the profile's entries that sit under a folder starting with a dot, which SMAPI
// skips; it lists none unless the game setting showDotHiddenMods is on.
func (s *Service) DotHiddenMods(game, id string) ([]HiddenMod, error) {
	if !s.settings.Get().GamePrefs(game).ShowDotHiddenMods {
		return []HiddenMod{}, nil
	}
	return s.store.DotHiddenMods(game, id)
}

func hiddenUnder(root string, ours map[string]bool) ([]HiddenMod, error) {
	base, err := filepath.EvalSymlinks(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []HiddenMod
	var walk func(dir, rel string, hidden bool) error
	walk = func(dir, rel string, hidden bool) error {
		b, err := fsx.ReadFile(filepath.Join(dir, manifest.FileName))
		if err == nil {
			if m, perr := manifest.Parse(b); perr == nil && hidden {
				out = append(out, HiddenMod{Folder: rel, UniqueID: m.UniqueID, Name: m.Name, Author: m.Author, Version: m.Version})
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		items, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, it := range items {
			child := filepath.Join(dir, it.Name())
			if !datadir.RealDirUnder(base, child) {
				continue
			}
			next := path.Join(rel, it.Name())
			dotted := strings.HasPrefix(it.Name(), ".") && !ours[next]
			if err := walk(child, next, hidden || dotted); err != nil {
				return err
			}
		}
		return nil
	}
	return out, walk(root, ".", false)
}
