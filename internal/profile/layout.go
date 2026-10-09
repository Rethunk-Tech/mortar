package profile

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fomod"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// driverThunderstore is the installer driver for Thunderstore packages, whose files go to the profile target and are
// deployed at launch instead of being copied into the profile's mods folder.
const driverThunderstore = "thunderstore-rules"

// installerGame is the game as the installer sees it: the loaders it runs and the content targets the catalog gives it.
func installerGame(gameID string) installer.Game {
	info, _ := components.Game(gameID)
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
	if _, statErr := os.Stat(filepath.Join(root, "manifest.json")); inst.ID() == driverThunderstore && statErr == nil {
		// The driver names files by package, Namespace-Name, which is not the store key. An archive from disk has only
		// the manifest's name, which the driver reads itself. A BepInEx mod without a manifest keeps the store key.
		_, arch.Key, _, _ = s.items.Meta(game, key)
		// A GitHub item is described by owner/repo, which may hold a dash but is no package name.
		if _, _, isTS := strings.Cut(arch.Key, "-"); !isTS || strings.Contains(arch.Key, "/") {
			arch.Key = ""
		}
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
	l, err := inst.Layout(arch, g, choices)
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
func (e Entry) hasFolder() bool { return !e.Package }

// namePackage records the Namespace-Name of a Thunderstore package added from disk when its manifest does not carry a
// namespace: from the download's file name, which Thunderstore makes <Namespace>-<Name>-<version>.zip, else from the
// one package of that name and version in the game's index, else fallback (a GitHub release's owner). Nothing is
// recorded when none names it; packageMods then refuses the package.
func (s *Store) namePackage(game, key, fileName, fallback string) error {
	arch, _, inst, err := s.pick(game, key)
	if err != nil || inst.ID() != driverThunderstore || arch.Key != "" {
		return err
	}
	b, err := fsx.ReadFile(filepath.Join(arch.Dir, "manifest.json"))
	if err != nil {
		return err
	}
	m, err := bepinex5.ParseManifest(b)
	if err != nil {
		return err
	}
	if m.Namespace != "" || m.Author != "" {
		return nil
	}
	ns := ""
	stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if head, ok := cutSuffixFold(stem, "-"+m.Name+"-"+m.Version); ok && head != "" && !strings.Contains(head, "-") {
		ns = head
	} else if s.Publisher != nil {
		ns, _ = s.Publisher(game, m.Name, m.Version)
	}
	ns = cmp.Or(ns, fallback)
	if ns == "" {
		return nil
	}
	return s.items.Describe(game, key, KindThunderstore, ns+"-"+m.Name, m.Version, "")
}

func cutSuffixFold(s, suffix string) (string, bool) {
	if len(s) < len(suffix) || !strings.EqualFold(s[len(s)-len(suffix):], suffix) {
		return "", false
	}
	return s[:len(s)-len(suffix)], true
}

// packageMods is the one component of a Thunderstore package item; ok is false for any other item.
func (s *Store) packageMods(game, key string) (mods []Component, ok bool, err error) {
	arch, g, inst, err := s.pick(game, key)
	if err == nil && inst.ID() == "plain" && slices.Contains(g.Loaders, folderLoader) {
		return []Component{{ID: mod.NewID(mod.FormatFolder, key), Name: key, Folder: "."}}, true, nil
	}
	if err != nil || inst.ID() != driverThunderstore {
		return nil, false, err
	}
	b, err := fsx.ReadFile(filepath.Join(arch.Dir, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return []Component{pluginComponent(key, arch.Dir)}, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	m, err := bepinex5.ParseManifest(b)
	if err != nil {
		return nil, false, err
	}
	author, _, _ := strings.Cut(arch.Key, "-")
	id := arch.Key
	if id == "" {
		// A package added from disk has no index entry, so its Namespace-Name comes from its manifest; Thunderstore
		// needs that to match the package against the index, so without it the archive is refused.
		author = cmp.Or(m.Namespace, m.Author)
		if author == "" {
			return nil, false, usererr.Wrap(usererr.Invalid, fmt.Errorf("%s does not say who published it, and neither its file name nor the Thunderstore index does, so Mortar cannot tell which package it is: install it from Thunderstore instead", m.Name))
		}
		id = author + "-" + m.Name
	}
	var needs []mod.ID
	for _, d := range m.Needs() {
		needs = append(needs, d.ModID())
	}
	return []Component{{ID: mod.NewID(mod.FormatThunderstore, id), Version: m.Version, Name: m.Name, Author: author, Folder: ".", Needs: needs}}, true, nil
}

// folderLoader is the id of the loader of a game whose mods are files in a folder the game reads (internal/loader/folder).
const folderLoader = "folder"

// pluginComponent is the component of a BepInEx mod that has no Thunderstore manifest. It is named for its first
// DLL, which is the plugin's assembly in nearly every such archive, so another file or version of the same mod
// replaces it; an archive with no DLL is named for its store key.
// It has no version of its own; the entry's source gives it one.
func pluginComponent(key, dir string) Component {
	name := key
	var dlls []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".dll") {
			dlls = append(dlls, p)
		}
		return nil
	})
	if len(dlls) > 0 {
		slices.Sort(dlls)
		name = strings.TrimSuffix(filepath.Base(dlls[0]), filepath.Ext(dlls[0]))
	}
	return Component{ID: mod.NewID(mod.FormatBepInEx, name), Name: name, Folder: "."}
}

// placePackageLocked adds a Thunderstore package to the profile, or swaps it in for another version of the same
// package. Nothing is copied: the entry records the package, and the launch deploys its files.
func (s *Store) placePackageLocked(game, id, key string, source Source, mods []Component) (Profile, bool, bool, error) {
	var updated, changed bool
	mods[0].Version = cmp.Or(mods[0].Version, source.Version)
	p, err := s.updateLocked(game, id, func(p *Profile, dir string) error {
		if i := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == key }); i >= 0 {
			return &DuplicateError{Key: key, Label: entryLabel(p.Entries[i])}
		}
		i := slices.IndexFunc(p.Entries, func(e Entry) bool {
			return e.Package && len(e.Mods) == 1 && mod.Equal(e.Mods[0].ID, mods[0].ID)
		})
		if i < 0 {
			_, err := s.addTo(game, p, dir, key, source, nil)
			return err
		}
		updated, changed = true, modsVersionChanged(p.Entries[i].Mods, mods)
		ne, err := s.swapPackage(game, id, p.Entries[i], key, mods)
		ne.Source = source
		p.Entries[i] = ne
		return err
	})
	if err != nil {
		return Profile{}, false, false, err
	}
	return p, updated, changed, s.items.Touch(game, key)
}

// swapPackage is package entry e pointed at newKey, another version of it holding mods, after the pre-update save
// backup. Nothing is copied: the launch deploys the package's files.
func (s *Store) swapPackage(game, id string, e Entry, newKey string, mods []Component) (Entry, error) {
	if err := s.saveBackup(game, id); err != nil {
		return Entry{}, fmt.Errorf("back up saves: %w", err)
	}
	prev := e.Source
	e.PreviousKey, e.PreviousSource = e.Key, &prev
	e.Key, e.Mods, e.SkipVersion = newKey, mods, ""
	return e, nil
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
