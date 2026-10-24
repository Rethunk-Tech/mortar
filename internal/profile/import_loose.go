package profile

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// looseModKinds is whether the game reads its mods as loose files from a folder (the folder loader) and the
// lower-case extensions, without a dot, that its mods target names; none named means any file is a mod.
func looseModKinds(game string) ([]string, bool) {
	g := installerGame(game)
	if !slices.Contains(g.Loaders, folderLoader) {
		return nil, false
	}
	var kinds []string
	for _, t := range g.Targets {
		if t.ID != installer.TargetMods {
			continue
		}
		for ext := range t.MaxDepth {
			kinds = append(kinds, ext)
		}
	}
	return kinds, true
}

// offSuffix is what players and mod tools add to a mod file's extension to switch it off without deleting it
// ("hair.packageoff").
const offSuffix = "off"

func extOf(name string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
}

// switchedOn is the name a switched-off mod file has when it is on; ok is false for any other file.
func switchedOn(name string, kinds []string) (on string, ok bool) {
	kind, off := strings.CutSuffix(extOf(name), offSuffix)
	if !off || !slices.Contains(kinds, kind) {
		return "", false
	}
	return name[:len(name)-len(offSuffix)], true
}

func isLooseMod(name string, kinds []string) bool {
	if _, off := switchedOn(name, kinds); off {
		return true
	}
	return len(kinds) == 0 || slices.Contains(kinds, extOf(name))
}

// switchStagedOn gives the switched-off mod files below staged their real names, since a profile records what is
// off itself, and returns their slash paths; all is true when the copy is one unit (it holds a file that only
// works beside its siblings) and every mod file in it was off. A unit that is only partly off keeps its off names,
// which the game goes on ignoring, because a unit is switched as one. An off file beside its on twin is dropped.
func switchStagedOn(staged string, kinds, keepWhole []string) (off map[string]bool, all bool, err error) {
	type file struct{ path, on string }
	var offFiles []file
	on, unit := 0, false
	err = filepath.WalkDir(staged, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name, isOff := switchedOn(d.Name(), kinds)
		if isOff {
			offFiles = append(offFiles, file{p, filepath.Join(filepath.Dir(p), name)})
		} else if name = d.Name(); isLooseMod(name, kinds) {
			on++
		}
		unit = unit || slices.Contains(keepWhole, extOf(name))
		return nil
	})
	if err != nil || len(offFiles) == 0 || (unit && on > 0) {
		return nil, false, err
	}
	off = map[string]bool{}
	for _, f := range offFiles {
		if _, statErr := os.Lstat(f.on); statErr == nil {
			if err := fsx.Remove(f.path); err != nil {
				return nil, false, err
			}
			continue
		}
		if err := fsx.Rename(f.path, f.on); err != nil {
			return nil, false, err
		}
		rel, err := filepath.Rel(staged, f.on)
		if err != nil {
			return nil, false, err
		}
		off[filepath.ToSlash(rel)] = true
	}
	return off, unit, nil
}

// scanLooseMods lists what a folder game's mods folder holds at its top level as one slot each: a mod file, or a
// folder that holds one at any depth. Links and the game's own files (its Resource.cfg) are not mods.
func scanLooseMods(modsDir string, kinds []string) ([]gameModSlot, error) {
	entries, err := os.ReadDir(modsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	root, err := fsx.EvalSymlinks(modsDir)
	if err != nil {
		return nil, err
	}
	var slots []gameModSlot
	for _, it := range entries {
		child := filepath.Join(modsDir, it.Name())
		switch {
		case it.Type().IsRegular():
			if !isLooseMod(it.Name(), kinds) {
				continue
			}
		case datadir.RealDirUnder(root, child):
			if !holdsLooseMod(child, kinds) {
				continue
			}
		default:
			continue
		}
		_, off := switchedOn(it.Name(), kinds)
		slots = append(slots, gameModSlot{folder: gameModFolder{dir: child, label: it.Name(), loose: true, disabled: off}, ready: true})
	}
	return slots, nil
}

func holdsLooseMod(dir string, kinds []string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() && isLooseMod(d.Name(), kinds) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

// importLoose adds a folder game's mod file or folder to the profile under its own name, so it lays out in the
// profile where it sat in the game's folder.
func (s *Store) importLoose(ctx context.Context, game, id string, f gameModFolder) (key, change string, err error) {
	scratch, err := os.MkdirTemp(filepath.Dir(s.root), "mortar-import-*")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = fsx.RemoveAll(scratch) }()
	staged := filepath.Join(scratch, "mod")
	dest := filepath.Join(staged, f.label)
	info, err := os.Stat(f.dir)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		err = datadir.CopyTreeResolvingLinks(f.dir, dest)
	} else if err = os.MkdirAll(staged, 0o700); err == nil {
		err = datadir.CopyFile(f.dir, dest)
	}
	if err != nil {
		return "", "", err
	}
	kinds, _ := looseModKinds(game)
	var keepWhole []string
	for _, t := range installerGame(game).Targets {
		if t.ID == installer.TargetMods {
			keepWhole = t.KeepWhole
		}
	}
	off, allOff, err := switchStagedOn(staged, kinds, keepWhole)
	if err != nil {
		return "", "", err
	}
	if key, err = s.items.AddHashedDir(ctx, game, staged); err != nil {
		return "", "", err
	}
	res, err := s.installKey(game, id, key, Source{Kind: KindLocal, Name: f.label})
	if err != nil {
		return "", "", err
	}
	var refs []EnableRef
	for _, e := range res.Profile.Entries {
		if e.StoreKey() == key && len(e.Mods) > 0 && (allOff || off[e.File]) {
			refs = append(refs, EnableRef{Key: e.Key, ID: e.Mods[0].ID})
		}
	}
	if len(refs) == 0 {
		return key, res.Profile.LastChange, nil
	}
	p, err := s.SetModsEnabled(game, id, refs, false)
	return key, p.LastChange, err
}
