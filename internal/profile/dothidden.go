package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/usererr"
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

// DotHiddenMods lists the dot-hidden mods in the profile's user entries, or only in the entry with the given key
// when key is not empty.
func (s *Store) DotHiddenMods(game, id, key string) ([]HiddenMod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	modsDir := filepath.Join(dir, "mods")
	out := []HiddenMod{}
	for _, e := range p.Entries {
		if e.Source.Bundled() || (key != "" && e.Key != key) {
			continue
		}
		found, err := hiddenUnder(entryRoot(modsDir, e.Key), oursOf(e))
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

// entryRoot is the entry's folder in mods/, which carries a dot while the entry is switched off.
func entryRoot(modsDir, key string) string {
	root := filepath.Join(modsDir, key)
	if !exists(root) {
		root = filepath.Join(modsDir, "."+key)
	}
	return root
}

// oursOf is the folders Mortar dotted to switch the entry's mods off; they are its own, not hidden.
func oursOf(e Entry) map[string]bool {
	ours := map[string]bool{}
	for _, m := range e.Mods {
		if m.Folder != "." {
			ours[path.Join(path.Dir(m.Folder), "."+path.Base(m.Folder))] = true
		}
	}
	return ours
}

// DotHiddenMods lists mods inside the profile's entries that sit under a folder starting with a dot, which SMAPI
// skips; it lists none unless the game setting showDotHiddenMods is on. A non-empty key limits it to that entry.
func (s *Service) DotHiddenMods(game, id, key string) ([]HiddenMod, error) {
	if !s.settings.Get().GamePrefs(game).ShowDotHiddenMods {
		return []HiddenMod{}, nil
	}
	return s.store.DotHiddenMods(game, id, key)
}

// ShowHiddenMod opens the folder of the dot-hidden mod at folder (as DotHiddenMods lists it) in entry key.
func (s *Service) ShowHiddenMod(game, id, key, folder string) error {
	dir, err := s.store.hiddenModPath(game, id, key, folder)
	if err != nil {
		return err
	}
	return datadir.Open(dir)
}

// UnhideMod renames the dotted folders above the dot-hidden mod at folder (as DotHiddenMods lists it) in entry key
// so SMAPI loads it, and adds it to the entry's mods.
func (s *Service) UnhideMod(game, id, key, folder string) (Profile, error) {
	return s.store.UnhideMod(game, id, key, folder)
}

func (s *Store) hiddenModPath(game, id, key, folder string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return "", err
	}
	ei := entryIndex(p.Entries, key)
	if ei < 0 {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("no entry %s", key))
	}
	root := entryRoot(filepath.Join(dir, "mods"), key)
	if _, err := findHidden(root, p.Entries[ei], folder); err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(folder)), nil
}

func findHidden(root string, e Entry, folder string) (HiddenMod, error) {
	found, err := hiddenUnder(root, oursOf(e))
	if err != nil {
		return HiddenMod{}, err
	}
	i := slices.IndexFunc(found, func(h HiddenMod) bool { return h.Folder == folder })
	if i < 0 {
		return HiddenMod{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("no hidden mod in %s", folder))
	}
	return found[i], nil
}

// UnhideMod strips the leading dots from every folder on the path to the hidden mod. It refuses when the mod's
// UniqueID is already loaded elsewhere in the profile or a folder with the undotted name exists.
func (s *Store) UnhideMod(game, id, key, folder string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	return s.updateLocked(game, id, func(p *Profile, dir string) error {
		ei := entryIndex(p.Entries, key)
		if ei < 0 {
			return usererr.Wrap(usererr.NotFound, fmt.Errorf("no entry %s", key))
		}
		e := &p.Entries[ei]
		root := entryRoot(filepath.Join(dir, "mods"), key)
		h, err := findHidden(root, *e, folder)
		if err != nil {
			return err
		}
		for _, other := range p.Entries {
			if slices.ContainsFunc(other.Mods, func(m EntryMod) bool { return SameID(m.UniqueID, h.UniqueID) }) {
				return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is already loaded from another folder", h.Name))
			}
		}
		newRel, err := stripDots(root, folder, oursOf(*e))
		if err != nil {
			return err
		}
		b, err := fsx.ReadFile(filepath.Join(root, filepath.FromSlash(newRel), manifest.FileName))
		if err != nil {
			return err
		}
		m, err := manifest.Parse(b)
		if err != nil {
			return err
		}
		e.Mods = append(e.Mods, entryMods([]manifest.Mod{{Manifest: m, Folder: newRel}})...)
		return nil
	})
}

// stripDots renames each folder of rel, a path under root, whose name starts with a dot Mortar did not add, and
// returns the new path.
func stripDots(root, rel string, ours map[string]bool) (string, error) {
	cur, orig, renamed := root, "", ""
	for name := range strings.SplitSeq(rel, "/") {
		orig = path.Join(orig, name)
		next := name
		if strings.HasPrefix(name, ".") && !ours[orig] {
			next = strings.TrimLeft(name, ".")
			if next == "" || exists(filepath.Join(cur, next)) {
				return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("cannot rename %s: %q is taken", name, next))
			}
			if err := fsx.Rename(filepath.Join(cur, name), filepath.Join(cur, next)); err != nil {
				return "", err
			}
		}
		cur = filepath.Join(cur, next)
		renamed = path.Join(renamed, next)
	}
	return renamed, nil
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
