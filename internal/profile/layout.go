package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fomod"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// driverThunderstore is the installer driver for Thunderstore packages, whose files go to the profile target and are
// deployed at launch instead of being copied into the profile's mods folder.
const driverThunderstore = "thunderstore-rules"

// installerGame is the game as the installer sees it: the loaders it runs and the content targets the catalog gives it.
func installerGame(gameID string) installer.Game {
	info, _ := components.BundledGame(gameID)
	var g installer.Game
	for _, l := range info.Loaders {
		g.Loaders = append(g.Loaders, l.ID)
	}
	for _, t := range info.Targets {
		g.Targets = append(g.Targets, installer.Target{ID: t.ID, Root: t.Root, MaxDepth: t.MaxDepth})
	}
	return g
}

// pick is the archive of the store item key and the installer that takes it.
func (s *Store) pick(game, key string) (installer.Archive, installer.Game, installer.Installer, error) {
	root, err := s.items.Path(game, key)
	if err != nil {
		return installer.Archive{}, installer.Game{}, nil, err
	}
	arch := installer.Open(root, key)
	g := installerGame(game)
	inst, _ := installer.Pick(arch, g)
	if inst.ID() == driverThunderstore {
		// The driver names files by package, Namespace-Name, which is not the store key.
		_, arch.Key, _, _ = s.items.Meta(game, key)
	}
	return arch, g, inst, nil
}

// layoutOf picks the installer for the store item key and lays it out: the archive (the item's chosen mod root), the
// files and where they go, and the driver's id. A FOMOD whose choices are missing or stale is a NeedChoicesError.
func (s *Store) layoutOf(game, id, key string, choices map[string]map[string][]string) (installer.Archive, installer.Layout, string, error) {
	arch, g, inst, err := s.pick(game, key)
	if err != nil {
		return installer.Archive{}, installer.Layout{}, "", err
	}
	root := arch.Dir
	if inst.ID() == "fomod" {
		cfg, _, _, err := fomod.Open(root)
		if err != nil {
			return arch, installer.Layout{}, "", err
		}
		modsDir, err := s.ModsDir(game, id)
		if err != nil {
			return arch, installer.Layout{}, "", err
		}
		arch.Eval = s.fomodEval(game, s.fileIndex(modsDir))
		if !fomod.Match(cfg, choices, arch.Eval) {
			return arch, installer.Layout{}, "", &NeedChoicesError{Ask: askFrom(cfg, key, Source{}, "", choices, arch.Eval)}
		}
	}
	l, err := inst.Layout(context.Background(), arch, g, choices)
	return arch, l, inst.ID(), err
}

// writeLayout puts the layout's files in dst as the item's own folder, so a file at <key>/a/b lands at dst/a/b.
func writeLayout(arch installer.Archive, l installer.Layout, dst string) error {
	for _, f := range l.Files {
		to := filepath.Join(dst, filepath.FromSlash(strings.TrimPrefix(f.Rel, arch.Key+"/")))
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return err
		}
		if err := datadir.MaterializeFile(filepath.Join(arch.Dir, filepath.FromSlash(f.Src)), to, f.Rel); err != nil {
			return err
		}
	}
	return nil
}

// layoutItem is a folder holding the store item as its entry lays out. A plain item is its own folder; any other is
// written to a temp folder, tmp, that the caller removes.
func (s *Store) layoutItem(game, id, key string, choices map[string]map[string][]string) (src, tmp string, err error) {
	arch, l, driver, err := s.layoutOf(game, id, key, choices)
	if err != nil || driver == "plain" {
		return arch.Dir, "", err
	}
	tmp, err = os.MkdirTemp("", "mortar-fomod-")
	if err != nil {
		return "", "", err
	}
	if err := writeLayout(arch, l, tmp); err != nil {
		_ = fsx.RemoveAll(tmp)
		return "", "", err
	}
	return tmp, tmp, nil
}

// hasFolder reports an entry that has its own folder in the profile's mods folder. A Thunderstore package has none:
// its files go to the profile target when the game launches.
func (e Entry) hasFolder() bool { return e.Source.Kind != KindThunderstore }

// packageMods is the one component of a Thunderstore package item; ok is false for any other item.
func (s *Store) packageMods(game, key string) (mods []Component, ok bool, err error) {
	arch, _, inst, err := s.pick(game, key)
	if err != nil || inst.ID() != driverThunderstore {
		return nil, false, err
	}
	var m struct {
		Name    string `json:"name"`
		Version string `json:"version_number"`
	}
	b, err := fsx.ReadFile(filepath.Join(arch.Dir, "manifest.json"))
	if err != nil {
		return nil, false, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false, err
	}
	author, _, _ := strings.Cut(arch.Key, "-")
	return []Component{{ID: mod.NewID(mod.FormatThunderstore, arch.Key), Version: m.Version, Name: m.Name, Author: author, Folder: "."}}, true, nil
}

// placePackageLocked adds a Thunderstore package to the profile, or swaps it in for another version of the same
// package. Nothing is copied: the entry records the package, and the launch deploys its files.
func (s *Store) placePackageLocked(game, id, key string, source Source, mods []Component) (Profile, bool, bool, error) {
	var updated, changed bool
	p, err := s.updateLocked(game, id, func(p *Profile, dir string) error {
		if i := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == key }); i >= 0 {
			return &DuplicateError{Key: key, Label: entryLabel(p.Entries[i])}
		}
		i := slices.IndexFunc(p.Entries, func(e Entry) bool {
			return e.Source.Kind == KindThunderstore && strings.EqualFold(e.Source.Name, source.Name)
		})
		if i < 0 {
			_, err := s.addTo(game, p, dir, key, source, nil)
			return err
		}
		old := p.Entries[i]
		updated, changed = true, modsVersionChanged(old.Mods, mods)
		prev := old.Source
		old.PreviousKey, old.PreviousSource = old.Key, &prev
		old.Key, old.Source, old.Mods = key, source, mods
		p.Entries[i] = old
		return nil
	})
	if err != nil {
		return Profile{}, false, false, err
	}
	return p, updated, changed, s.items.Touch(game, key)
}

// WriteFiles writes files into the profile, by slash path relative to its folder. A path that leaves the folder is
// refused before anything is written.
func (s *Store) WriteFiles(game, id string, files map[string][]byte) error {
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	_, dir, err := s.readDir(game, id)
	if err != nil {
		return err
	}
	for rel := range files {
		if !filepath.IsLocal(filepath.FromSlash(rel)) {
			return fmt.Errorf("pack file %q leaves the profile", rel)
		}
	}
	for rel, data := range files {
		to := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return err
		}
		if err := fsx.WriteFile(to, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}
