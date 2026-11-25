package profile

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	ga "github.com/Rethunk-Tech/go-archive"

	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// fileKeySep joins a store item's key and a file's path into the key of a per-file entry; no store key holds it.
const fileKeySep = "#"

// folderEntries are the entries a folder-loader game's store item key becomes: one per laid-out file, or the one
// entry holding them all when a file in it only works beside its siblings. Each keeps the archive's source.
func (s *Store) folderEntries(game, id, key string, source Source, whole []Component) (entries []Entry, tray []installer.File, arch installer.Archive, err error) {
	arch, l, _, err := s.layoutOf(game, id, key, nil)
	if err != nil {
		return nil, nil, arch, err
	}
	if err := s.checkUnknownLayout(game, id, key, l); err != nil {
		return nil, nil, arch, err
	}
	if err := s.recordInstallNotes(game, id, key, whole, l); err != nil {
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
		// A file of the player's own has no page to take a name from, and its store key is a hash.
		if source.Kind == KindLocal && source.Name != "" && len(whole) == 1 && whole[0].Name == key {
			whole = []Component{whole[0]}
			whole[0].Name = ga.StripExtension(source.Name)
		}
		return []Entry{{Key: key, Source: source, Mods: whole, Disabled: []mod.ID{}, Added: now, Package: true, Implicit: source.startsImplicit()}}, tray, arch, nil
	}
	out := make([]Entry, 0, len(mods.Files))
	for _, f := range mods.Files {
		k := key + fileKeySep + f.Rel
		out = append(out, Entry{
			Key: k, Item: key, File: f.Rel, Source: source, Disabled: []mod.ID{}, Added: now, Package: true, Implicit: source.startsImplicit(),
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
		if err := duplicatePackage(p.Entries, key); err != nil {
			return err
		}
		off := map[string]bool{}
		var old []Entry
		p.Entries = slices.DeleteFunc(p.Entries, func(e Entry) bool {
			if e.IsOverlay() || !e.Package || !replaces(source, e.Source) || e.StoreKey() == key {
				return false
			}
			old = append(old, e)
			off[e.File] = !e.HasPackageEnabled()
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
				replace[f.folded()] = f.Hash
			}
		}
		placement, err = s.placeTray(game, p, arch, trayFiles, replace)
		if err != nil {
			return err
		}
		if held := placement.held; len(held) > 0 {
			fresh = append(fresh, trayHoldEntry(key, source, held))
		}
		disableFresh(fresh, off, source)
		carryPlayerState(fresh, old)
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

// duplicatePackage is the error for a package entry of store item key already in entries.
func duplicatePackage(entries []Entry, key string) error {
	for _, e := range entries {
		if e.StoreKey() == key && e.Package && !e.IsOverlay() {
			return &DuplicateError{Key: e.Key, Label: entryLabel(e)}
		}
	}
	return nil
}

// trayHoldEntry is the entry that holds the Tray files an install placed.
func trayHoldEntry(key string, source Source, held []TrayFile) Entry {
	k := trayEntryKey(key)
	name := "Tray files"
	if held[0].Role != "" {
		name = cmp.Or(source.ModName, key)
	}
	return Entry{
		Key: k, Item: key, Source: source, Disabled: []mod.ID{}, Added: time.Now().UTC(), Package: true, Tray: true, TrayFiles: held, Implicit: source.startsImplicit(),
		Mods: []Component{{ID: mod.NewID(mod.FormatFolder, k), Name: name, Folder: "."}},
	}
}

// disableFresh switches off the new components of a file that was off before the update, or that the source says is off.
func disableFresh(fresh []Entry, off map[string]bool, source Source) {
	allOff := len(off) > 0 && !slices.Contains(slices.Collect(maps.Values(off)), false)
	for i := range fresh {
		// A file the old version did not have is off when the player had switched the whole download off.
		wasOff, known := off[fresh[i].File]
		if !known {
			wasOff = allOff
		}
		for _, m := range fresh[i].Mods {
			if wasOff || (source.disabled != nil && hasID(source.disabled.ids, m.ID)) {
				fresh[i].Disabled = append(fresh[i].Disabled, m.ID)
			}
		}
	}
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
		i := EntryIndex(p.Entries, key)
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
			want[f.folded()] = true
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
		if g.IsShared(f.Target) && want[TrayFile{Rel: f.Rel, Role: trayFileRole(g.RoleOf(f.Target))}.folded()] {
			files = append(files, f)
		}
	}
	replace := map[string]string{}
	for _, e := range current {
		for _, f := range e.TrayFiles {
			replace[f.folded()] = f.Hash
		}
	}
	pl, err := s.placeTray(game, p, arch, files, replace)
	if err != nil {
		return err
	}
	*out = pl
	for i := range restored {
		if restored[i].Tray {
			restored[i].TrayFiles = pl.held
			break
		}
	}
	return nil
}

// RepairFolderEntries brings a folder-loader game's entries that older releases saved up to how an install records
// them now. A download split one entry per file whose files no longer match what the game's catalog lays out (the
// catalog now keeps it whole, or its layout driver moved its files) is laid out again, the way an update would, keeping
// what the player set on its parts and switched off only when every part it replaces was. A split download whose
// parts all still lay out is left as it is, even with some removed. A player's own archive named with its file ending
// loses it.
func (s *Store) RepairFolderEntries(game, id string) (bool, error) {
	g := installerGame(game)
	ti := slices.IndexFunc(g.Targets, func(t installer.Target) bool { return t.ID == installer.TargetMods })
	if !slices.Contains(g.Loaders, folderLoader) || ti < 0 {
		return false, nil
	}
	p, err := s.Get(game, id)
	if err != nil {
		return false, err
	}
	keys := staleSplitItems(p.Entries, g.Targets[ti])
	if len(keys) == 0 && !slices.ContainsFunc(p.Entries, namedWithEnding) {
		return false, nil
	}
	if err := s.lockUnlocked(game, id); err != nil {
		return false, err
	}
	defer s.mu.Unlock()
	relaid := map[string][]Entry{}
	for _, key := range keys {
		parts := splitParts(p.Entries, key)
		mods, _, err := s.packageMods(game, key, parts[0].Source.ModName)
		if err != nil {
			continue
		}
		// The download is already installed, so the question a new install of an odd layout asks is not asked again.
		s.layoutOK.allow(game, id, key)
		fresh, _, _, err := s.folderEntries(game, id, key, parts[0].Source, mods)
		s.layoutOK.take(game, id, key)
		if err != nil || len(fresh) == 0 || stillLaidOut(parts, fresh) {
			continue
		}
		relaid[key] = fresh
	}
	if len(relaid) == 0 && !slices.ContainsFunc(p.Entries, namedWithEnding) {
		return false, nil
	}
	_, err = s.updateLocked(game, id, func(p *Profile, _ string) error {
		p.Entries = relaidEntries(p.Entries, relaid)
		for i := range p.Entries {
			if namedWithEnding(p.Entries[i]) {
				p.Entries[i].Mods = slices.Clone(p.Entries[i].Mods)
				p.Entries[i].Mods[0].Name = ga.StripExtension(p.Entries[i].Source.Name)
			}
		}
		return nil
	})
	return err == nil, err
}

// namedWithEnding reports a whole entry of the player's own archive whose mod is named for the file, ending and all.
func namedWithEnding(e Entry) bool {
	return e.Source.Kind == KindLocal && e.Package && e.File == "" && !e.IsOverlay() && len(e.Mods) == 1 &&
		e.Mods[0].Name == e.Source.Name && ga.StripExtension(e.Source.Name) != e.Source.Name
}

// staleSplitItems are the store keys of split downloads that may no longer match their layout: every one under a
// layout driver, which can move files between releases, and any holding a file the mods target now keeps whole.
func staleSplitItems(entries []Entry, t installer.Target) []string {
	var keys []string
	for _, e := range entries {
		if e.Item == "" || e.File == "" || e.IsOverlay() || slices.Contains(keys, e.Item) {
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(e.File), "."))
		if t.Layout != "" || slices.Contains(t.KeepWhole, ext) || t.ModRoot != "" && strings.EqualFold(path.Base(e.File), t.ModRoot) {
			keys = append(keys, e.Item)
		}
	}
	return keys
}

// splitParts are the per-file entries of store item key, in profile order.
func splitParts(entries []Entry, key string) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Item == key && e.File != "" && !e.IsOverlay() {
			out = append(out, e)
		}
	}
	return out
}

// stillLaidOut reports split parts that each name a file the fresh layout still splits out.
func stillLaidOut(parts, fresh []Entry) bool {
	return !slices.ContainsFunc(parts, func(p Entry) bool {
		return !slices.ContainsFunc(fresh, func(f Entry) bool { return f.File != "" && f.File == p.File })
	})
}

// relaidEntries is entries with the parts of each download in relaid replaced by its fresh entries, at the first
// part's place.
func relaidEntries(entries []Entry, relaid map[string][]Entry) []Entry {
	out := make([]Entry, 0, len(entries))
	done := map[string]bool{}
	for _, e := range entries {
		fresh, ok := relaid[e.Item]
		if !ok || e.File == "" || e.IsOverlay() {
			out = append(out, e)
			continue
		}
		if done[e.Item] {
			continue
		}
		done[e.Item] = true
		parts := splitParts(entries, e.Item)
		fresh = slices.Clone(fresh)
		carryPlayerState(fresh, parts)
		for i := range fresh {
			if !slices.ContainsFunc(partsFor(fresh[i], parts), Entry.HasPackageEnabled) {
				for _, m := range fresh[i].Mods {
					fresh[i].Disabled = append(fresh[i].Disabled, m.ID)
				}
			}
		}
		out = append(out, fresh...)
	}
	return out
}

// partsFor are the old entries a fresh one replaces: those laying out its file, else all of them.
func partsFor(fresh Entry, old []Entry) []Entry {
	same := slices.DeleteFunc(slices.Clone(old), func(o Entry) bool { return o.File != fresh.File })
	if len(same) == 0 {
		return old
	}
	return same
}

// carryPlayerState gives each fresh entry what the player set on the old entries it replaces (partsFor), merged so
// a flag any of them had is kept and the first text set wins.
func carryPlayerState(fresh, old []Entry) {
	for i := range fresh {
		d := &fresh[i]
		for _, o := range partsFor(*d, old) {
			d.Pinned, d.IgnoreUpdates, d.DevOnly = d.Pinned || o.Pinned, d.IgnoreUpdates || o.IgnoreUpdates, d.DevOnly || o.DevOnly
			d.PinReason, d.Note, d.UpdateChannel = cmp.Or(d.PinReason, o.PinReason), cmp.Or(d.Note, o.Note), cmp.Or(d.UpdateChannel, o.UpdateChannel)
			d.DisplayName, d.DisplayAuthor = cmp.Or(d.DisplayName, o.DisplayName), cmp.Or(d.DisplayAuthor, o.DisplayAuthor)
			d.DisplayVersion, d.CategoryOverride = cmp.Or(d.DisplayVersion, o.DisplayVersion), cmp.Or(d.CategoryOverride, o.CategoryOverride)
			d.REDmod = cmp.Or(d.REDmod, o.REDmod)
			d.Tags = appendMissing(d.Tags, o.Tags)
			d.SkipSources = appendMissing(d.SkipSources, o.SkipSources)
			if d.Fomod == nil {
				d.Fomod = o.Fomod
			}
			if !o.Added.IsZero() && (d.Added.IsZero() || o.Added.Before(d.Added)) {
				d.Added = o.Added
			}
		}
	}
}

func appendMissing(to, from []string) []string {
	for _, v := range from {
		if !slices.Contains(to, v) {
			to = append(to, v)
		}
	}
	return to
}
