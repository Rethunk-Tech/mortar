package profile

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// fileKeySep joins a store item's key and a file's path into the key of a per-file entry; no store key holds it.
const fileKeySep = "#"

// supersedes reports whether the install of next is the update of the archive installed from old: a page's file
// whose update names old's file id (Source.WithReplacing), or a repository's release asset of the same shape. A fresh
// install, another file of the same page and a local archive replace nothing.
func supersedes(next, old Source) bool {
	if next.Kind != old.Kind {
		return false
	}
	switch next.Kind {
	case KindGitHub:
		return next.Repo != "" && strings.EqualFold(next.Repo, old.Repo) && next.Tag != old.Tag &&
			github.Shape(next.Asset) == github.Shape(old.Asset)
	case KindLocal:
		return false
	}
	return next.replacing > 0 && next.ModID == old.ModID && old.FileID == next.replacing
}

// folderEntries are the entries a folder-loader game's store item key becomes: one per laid-out file, or the one
// entry holding them all when a file in it only works beside its siblings. Each keeps the archive's source.
func (s *Store) folderEntries(game, id, key string, source Source, whole []Component) (entries []Entry, tray []installer.File, arch installer.Archive, err error) {
	arch, l, _, err := s.layoutOf(game, id, key, nil)
	if err != nil {
		return nil, nil, arch, err
	}
	g := installerGame(game)
	var mods installer.Layout
	for _, f := range l.Files {
		if g.IsShared(f.Target) {
			tray = append(tray, f)
		} else {
			mods.Files = append(mods.Files, f)
		}
	}
	now := time.Now().UTC()
	if len(mods.Files) == 0 {
		return nil, tray, arch, nil
	}
	if !g.Splits(mods) {
		return []Entry{{Key: key, Source: source, Mods: whole, Disabled: []mod.ID{}, Added: now, Package: true}}, tray, arch, nil
	}
	out := make([]Entry, 0, len(mods.Files))
	for _, f := range mods.Files {
		k := key + fileKeySep + f.Rel
		out = append(out, Entry{
			Key: k, Item: key, File: f.Rel, Source: source, Disabled: []mod.ID{}, Added: now, Package: true,
			Mods: []Component{{ID: mod.NewID(mod.FormatFolder, k), Name: f.Rel, Folder: "."}},
		})
	}
	return out, tray, arch, nil
}

// placeFolderLocked adds a folder-loader game's store item to the profile as folderEntries says. The entries of
// the archive this install updates are all replaced together, each file keeping its on or off state. Files bound for
// the shared Tray folder are placed now and held by an entry of their own.
func (s *Store) placeFolderLocked(game, id, key string, source Source, whole []Component) (Profile, bool, bool, error) {
	fresh, trayFiles, arch, err := s.folderEntries(game, id, key, source, whole)
	if err != nil {
		return Profile{}, false, false, err
	}
	var updated bool
	placement := noTray()
	var released []TrayFile
	p, err := s.updateLocked(game, id, func(p *Profile, _ string) error {
		for _, e := range p.Entries {
			if e.StoreKey() == key && e.Package && !e.IsOverlay() {
				return &DuplicateError{Key: e.Key, Label: entryLabel(e)}
			}
		}
		off := map[string]bool{}
		var old []Entry
		p.Entries = slices.DeleteFunc(p.Entries, func(e Entry) bool {
			if e.IsOverlay() || !e.Package || !supersedes(source, e.Source) || e.StoreKey() == key {
				return false
			}
			old = append(old, e)
			off[e.File] = !e.hasPackageEnabled()
			return true
		})
		if updated = len(old) > 0; updated {
			if err := s.saveBackup(game, id); err != nil {
				return fmt.Errorf("back up saves: %w", err)
			}
		}
		replace := map[string]string{}
		for _, e := range old {
			released = append(released, e.TrayFiles...)
			for _, f := range e.TrayFiles {
				replace[f.Rel] = f.Hash
			}
		}
		placement, err = s.placeTray(game, p, arch, trayFiles, replace)
		if err != nil {
			return err
		}
		if held := placement.held; len(held) > 0 {
			k := trayEntryKey(key)
			fresh = append(fresh, Entry{
				Key: k, Item: key, Source: source, Disabled: []mod.ID{}, Added: time.Now().UTC(), Package: true, TrayFiles: held,
				Mods: []Component{{ID: mod.NewID(mod.FormatFolder, k), Name: "Tray files", Folder: "."}},
			})
		}
		for i := range fresh {
			if off[fresh[i].File] {
				for _, m := range fresh[i].Mods {
					fresh[i].Disabled = append(fresh[i].Disabled, m.ID)
				}
			}
		}
		if len(old) > 0 && len(fresh) > 0 {
			fresh[0].Replaced = packReplaced(old)
		}
		p.Entries = append(p.Entries, fresh...)
		return nil
	})
	if err != nil {
		placement.undo()
		return Profile{}, false, false, err
	}
	placement.commit()
	s.releaseTray(game, &p, released)
	return p, updated, updated, s.items.Touch(game, key)
}

// forgetReplaced is entries with their own rollback record dropped, so records never nest.
func forgetReplaced(entries []Entry) []Entry {
	out := make([]Entry, len(entries))
	for i, e := range entries {
		out[i] = e.Clone()
		out[i].Replaced = ""
	}
	return out
}

func packReplaced(entries []Entry) string {
	b, err := json.Marshal(forgetReplaced(entries))
	if err != nil {
		return ""
	}
	return string(b)
}

// replacedEntries are the entries Replaced records; a record that does not read is none.
func (e Entry) replacedEntries() []Entry {
	var out []Entry
	if e.Replaced != "" {
		_ = json.Unmarshal([]byte(e.Replaced), &out)
	}
	return out
}

// rollBackFolderLocked restores the entries the archive of entry key superseded, and records the archive it removes
// so that a second roll back goes forward again. ok is false when the entry's archive has nothing to restore.
func (s *Store) rollBackFolderLocked(game, id, key string) (p Profile, ok bool, err error) {
	var restored, group []Entry
	placement := noTray()
	p, err = s.updateLocked(game, id, func(p *Profile, _ string) error {
		i := entryIndex(p.Entries, key)
		if i < 0 {
			return nil
		}
		item := p.Entries[i].StoreKey()
		inGroup := func(e Entry) bool { return e.StoreKey() == item && e.Package && !e.IsOverlay() }
		at := slices.IndexFunc(p.Entries, inGroup)
		for _, e := range p.Entries {
			if inGroup(e) {
				group = append(group, e)
			}
		}
		g := slices.IndexFunc(group, func(e Entry) bool { return e.Replaced != "" })
		if g < 0 {
			return nil
		}
		prev := group[g].replacedEntries()
		if len(prev) == 0 {
			return nil
		}
		if err := s.saveBackup(game, id); err != nil {
			return fmt.Errorf("back up saves: %w", err)
		}
		p.Entries = slices.DeleteFunc(p.Entries, inGroup)
		if err := s.placeRestoredTray(game, id, p, prev, group, &placement); err != nil {
			return err
		}
		prev[0].Replaced = packReplaced(group)
		restored = prev
		p.Entries = slices.Insert(p.Entries, min(at, len(p.Entries)), restored...)
		return nil
	})
	if err != nil {
		placement.undo()
		return p, false, err
	}
	if restored == nil {
		return p, false, nil
	}
	placement.commit()
	var gone []TrayFile
	for _, e := range group {
		gone = append(gone, e.TrayFiles...)
	}
	s.releaseTray(game, &p, gone)
	return p, true, s.items.Touch(game, restored[0].StoreKey())
}

// placeRestoredTray puts back, from the store, the Tray files the restored entries held, replacing those of the
// entries leaving (current), and points each restored entry at what is now placed.
func (s *Store) placeRestoredTray(game, id string, p *Profile, restored, current []Entry, out *trayPlacement) error {
	want := map[string]bool{}
	item := ""
	for _, e := range restored {
		for _, f := range e.TrayFiles {
			want[f.Rel] = true
			item = e.StoreKey()
		}
	}
	if len(want) == 0 {
		return nil
	}
	arch, l, _, err := s.layoutOf(game, id, item, nil)
	if err != nil {
		return err
	}
	g := installerGame(game)
	var files []installer.File
	for _, f := range l.Files {
		if g.IsShared(f.Target) && want[f.Rel] {
			files = append(files, f)
		}
	}
	replace := map[string]string{}
	for _, e := range current {
		for _, f := range e.TrayFiles {
			replace[f.Rel] = f.Hash
		}
	}
	pl, err := s.placeTray(game, p, arch, files, replace)
	if err != nil {
		return err
	}
	*out = pl
	for i := range restored {
		if len(restored[i].TrayFiles) > 0 {
			restored[i].TrayFiles = pl.held
			break
		}
	}
	return nil
}
