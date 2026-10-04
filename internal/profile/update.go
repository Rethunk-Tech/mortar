package profile

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backup"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	gamepkg "github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// conflictSuffix names the profile's copy of a file the target version also changed.
const conflictSuffix = ".mortar-old"

// SpansEntriesError reports an archive whose mods sit in several entries of the profile, so no single one can be updated.
type SpansEntriesError struct{ Labels []string }

func (e *SpansEntriesError) Error() string {
	return fmt.Sprintf("the archive's mods are in %d entries of this profile (%v)", len(e.Labels), e.Labels)
}

// UpdateEntry replaces the entry oldKey with the store item newKey, carrying over what the user and the mod wrote.
func (s *Store) UpdateEntry(game, id, oldKey, newKey string) (Profile, error) {
	return s.moveTo(game, id, oldKey, newKey, nil)
}

// EntryMove is one entry switched to another store item by UpdateEntries.
type EntryMove struct {
	OldKey string `json:"oldKey"`
	NewKey string `json:"newKey"`
}

// UpdateEntries applies every move as one change: one history event, and no entry is switched unless all are. Moves
// repeated for the same entry, as a store item holding several mods yields, apply once; two different targets for
// one entry are refused.
func (s *Store) UpdateEntries(game, id string, moves []EntryMove) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	target := map[string]string{}
	var uniq []EntryMove
	for _, m := range moves {
		if prev, ok := target[m.OldKey]; ok {
			if prev != m.NewKey {
				return Profile{}, fmt.Errorf("%q cannot move to both %q and %q", m.OldKey, prev, m.NewKey)
			}
			continue
		}
		target[m.OldKey] = m.NewKey
		uniq = append(uniq, m)
	}
	var swaps []swapped
	p, err := s.updateLocked(game, id, func(p *Profile, dir string) error {
		for _, m := range uniq {
			ei, err := requireEntry(p.Entries, m.OldKey)
			if err != nil {
				return err
			}
			for _, e := range p.Entries {
				if e.Key == m.NewKey {
					return &DuplicateError{Key: m.NewKey, Label: entryLabel(e)}
				}
			}
			ne, w, err := s.swapOverlaid(game, p, dir, ei, m.NewKey, nil)
			swaps = append(swaps, w)
			if err != nil {
				return err
			}
			p.Entries[ei] = ne
		}
		return nil
	})
	if err != nil {
		for _, w := range slices.Backward(swaps) {
			err = errors.Join(err, w.undo())
		}
		return Profile{}, err
	}
	var keys []string
	for _, w := range swaps {
		w.commit()
	}
	if err := s.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	for _, m := range uniq {
		keys = append(keys, m.OldKey, m.NewKey)
		for _, e := range p.Entries {
			if e.Key == m.NewKey {
				keys = append(keys, e.ExtraStoreKeys...)
				keys = append(keys, e.PreviousExtraStoreKeys...)
			}
		}
	}
	return p, s.items.Touch(game, keys...)
}

// RollBack swaps the entry key back to its previous version, carrying over what the user and the mod wrote.
func (s *Store) RollBack(game, id, key string) (Profile, error) {
	return s.moveTo(game, id, key, "", nil)
}

// moveTo switches an entry to another store item. An empty newKey means the entry's previous key. A source
// replaces the entry's own.
func (s *Store) moveTo(game, id, oldKey, newKey string, source *Source) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	return s.moveToLocked(game, id, oldKey, newKey, source)
}

func (s *Store) moveToLocked(game, id, oldKey, newKey string, source *Source) (Profile, error) {
	var sw swapped
	p, err := s.updateLocked(game, id, func(p *Profile, dir string) error {
		ei, err := requireEntry(p.Entries, oldKey)
		if err != nil {
			return err
		}
		rollBack := newKey == ""
		if rollBack {
			if newKey = p.Entries[ei].PreviousKey; newKey == "" {
				return fmt.Errorf("%q has no previous version to roll back to", oldKey)
			}
		}
		for _, e := range p.Entries {
			if e.Key == newKey {
				return &DuplicateError{Key: newKey, Label: entryLabel(e)}
			}
		}
		ne, w, err := s.swapOverlaid(game, p, dir, ei, newKey, source)
		sw = w
		if err != nil {
			return err
		}
		switch {
		case source != nil:
			out := *source
			out.fomod = nil
			ne.Source = out
		case rollBack && p.Entries[ei].PreviousSource != nil:
			ne.Source = *p.Entries[ei].PreviousSource
		}
		p.Entries[ei] = ne
		return nil
	})
	if err != nil {
		return Profile{}, errors.Join(err, sw.undo())
	}
	sw.commit()
	if err := s.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	keys := []string{oldKey, newKey}
	for _, e := range p.Entries {
		if e.Key == newKey {
			keys = append(keys, e.ExtraStoreKeys...)
			keys = append(keys, e.PreviousExtraStoreKeys...)
			break
		}
	}
	return p, s.items.Touch(game, keys...)
}

// swapEntry builds the target version's folder beside mods/, carries over the profile's files, and renames it
// over the old one. It returns the entry as it stands afterwards, and the swap for the caller to commit once
// profile.json records it or undo if that fails.
func (s *Store) swapEntry(game, id, dir string, e Entry, newKey string, source *Source) (Entry, swapped, error) {
	choices, asker := e.Fomod, s.replayAsk
	if source != nil && source.fomod != nil {
		choices, asker = source.fomodMap(), s.fomodAsk
	}
	if ask, need, err := asker(game, id, newKey, e.Source, e.Key, choices); err != nil {
		return Entry{}, swapped{}, err
	} else if need {
		if source != nil {
			ask.Source = *source
			ask.Source.fomod = nil
		}
		return Entry{}, swapped{}, &NeedChoicesError{Ask: ask}
	}
	oldSrc, oldTmp, err := s.layoutItem(game, id, e.Key, e.Fomod)
	if oldTmp != "" {
		defer func() { _ = fsx.RemoveAll(oldTmp) }()
	}
	if err != nil {
		return Entry{}, swapped{}, err
	}
	newSrc, newTmp, err := s.layoutItem(game, id, newKey, choices)
	if newTmp != "" {
		defer func() { _ = fsx.RemoveAll(newTmp) }()
	}
	if err != nil {
		return Entry{}, swapped{}, err
	}
	found, err := manifest.Scan(newSrc)
	if err != nil {
		return Entry{}, swapped{}, err
	}
	if len(found) == 0 {
		return Entry{}, swapped{}, &NoModError{Key: newKey}
	}
	prevSource := e.Source
	ne := e
	ne.Key, ne.PreviousKey, ne.PreviousSource = newKey, e.Key, &prevSource
	if newKey == e.PreviousKey && len(e.PreviousExtraStoreKeys) > 0 {
		ne.ExtraStoreKeys = slices.Clone(e.PreviousExtraStoreKeys)
		ne.PreviousExtraStoreKeys = slices.Clone(e.ExtraStoreKeys)
	} else if len(e.ExtraStoreKeys) > 0 {
		ne.PreviousExtraStoreKeys = slices.Clone(e.ExtraStoreKeys)
		ne.ExtraStoreKeys = slices.Clone(e.ExtraStoreKeys)
	}
	ne.Mods, ne.Disabled, ne.SkipVersion = entryMods(found), []string{}, ""
	ne.Tags = slices.Clone(e.Tags)
	ne.Fomod = cloneFomod(choices)
	for _, m := range ne.Mods {
		if !e.Enabled(m.UniqueID) {
			ne.Disabled = append(ne.Disabled, m.UniqueID)
		}
	}
	if err := s.saveBackup(game, id); err != nil {
		return Entry{}, swapped{}, fmt.Errorf("back up saves: %w", err)
	}
	modsDir := filepath.Join(dir, "mods")
	if err := os.MkdirAll(modsDir, 0o700); err != nil {
		return Entry{}, swapped{}, err
	}
	tmp, err := os.MkdirTemp(modsDir, tempPrefix)
	if err != nil {
		return Entry{}, swapped{}, err
	}
	sw, err := fillUpdate(s, game, id, tmp, modsDir, oldSrc, newSrc, e, &ne, found)
	if err != nil {
		return Entry{}, swapped{}, errors.Join(err, fsx.RemoveAll(tmp))
	}
	return ne, sw, nil
}

func fillUpdate(s *Store, game, id, tmp, modsDir, oldSrc, newSrc string, e Entry, ne *Entry, newManifests []manifest.Mod) (swapped, error) {
	if err := datadir.MaterializeTree(newSrc, tmp); err != nil {
		return swapped{}, err
	}
	mode := s.oldFilesMode(game)
	var held []heldFile
	for _, nm := range ne.Mods {
		i := slices.IndexFunc(e.Mods, func(m EntryMod) bool { return SameID(m.UniqueID, nm.UniqueID) })
		if i < 0 {
			continue
		}
		plain, dotted, err := ModPaths(modsDir, e.Key, e.Mods[i].Folder)
		if err != nil {
			return swapped{}, err
		}
		cur := plain
		if !exists(cur) {
			cur = dotted
		}
		if !exists(cur) {
			continue
		}
		configOnly := deleteOldVersion(newManifests, nm.UniqueID)
		target := filepath.Join(tmp, filepath.FromSlash(nm.Folder))
		var gone func(rel, p string) error
		switch mode {
		case settings.OldFilesKeep:
			gone = func(rel, p string) error { return copyOver(p, filepath.Join(target, rel)) }
		case settings.OldFilesAsk:
			if _, err := safeFolder(nm.UniqueID); err == nil && filepath.Base(nm.UniqueID) == nm.UniqueID {
				gone = func(rel, p string) error {
					held = append(held, heldFile{uniqueID: nm.UniqueID, rel: rel, abs: p})
					return nil
				}
			}
		}
		err = carryOverWalk(cur, filepath.Join(oldSrc, filepath.FromSlash(e.Mods[i].Folder)), target, configOnly, gone)
		if err != nil {
			return swapped{}, err
		}
	}
	if len(ne.ExtraStoreKeys) > 0 {
		if err := s.fillExtrasUpdate(game, id, modsDir, e.Key, tmp, e, *ne); err != nil {
			return swapped{}, err
		}
		if err := s.refreshEntryMods(ne, tmp); err != nil {
			return swapped{}, err
		}
		ne.Disabled = ne.Disabled[:0]
		for _, m := range ne.Mods {
			if !e.Enabled(m.UniqueID) {
				ne.Disabled = append(ne.Disabled, m.UniqueID)
			}
		}
	}
	if err := applyLoadAfter(tmp, *ne, nil); err != nil {
		return swapped{}, err
	}
	final, err := materialize(tmp, *ne)
	if err != nil {
		return swapped{}, err
	}
	w, err := replaceFolder(modsDir, e.Key, tmp, final)
	if err != nil || w.aside == "" {
		return w, err
	}
	for _, h := range held {
		rel, err := filepath.Rel(w.old, h.abs)
		if err != nil {
			return w, err
		}
		w.held = append(w.held, heldFile{uniqueID: h.uniqueID, rel: h.rel, abs: filepath.Join(w.aside, rel)})
	}
	w.heldDir = filepath.Join(filepath.Dir(modsDir), oldFilesDir, ne.Key)
	return w, nil
}

// asidePrefix names an entry folder moved out of the way during an update. Its suffix is the folder's own name,
// so rebuild can put it back if Mortar stops before profile.json records the update.
const asidePrefix = tempPrefix + "aside_"

// swapped is an entry folder replaced on disk but not yet recorded in profile.json. held are files of the old folder
// to set aside under heldDir instead of deleting them with it.
type swapped struct {
	old, aside, placed string
	held               []heldFile
	heldDir            string
}

func (w swapped) commit() {
	// The store keeps the previous version, so a file that fails to move aside is still recoverable by Roll back.
	for _, h := range w.held {
		dst := filepath.Join(w.heldDir, h.uniqueID, h.rel)
		if os.MkdirAll(filepath.Dir(dst), 0o700) == nil {
			_ = fsx.Rename(h.abs, dst)
		}
	}
	if w.aside != "" {
		// A failed delete leaves a temp-prefixed folder, which the next rebuild sweeps.
		_ = fsx.RemoveAll(w.aside)
	}
}

func (w swapped) undo() error {
	if w.placed == "" {
		return nil
	}
	err := fsx.RemoveAll(w.placed)
	if w.aside != "" {
		err = errors.Join(err, fsx.Rename(w.aside, w.old))
	}
	return err
}

// replaceFolder moves the old entry folder aside and renames tmp into place, restoring the old folder if that fails.
func replaceFolder(modsDir, oldKey, tmp, final string) (swapped, error) {
	old := filepath.Join(modsDir, oldKey)
	if !exists(old) {
		old = filepath.Join(modsDir, "."+oldKey)
	}
	w := swapped{old: old, placed: filepath.Join(modsDir, final)}
	if exists(old) {
		w.aside = filepath.Join(modsDir, asidePrefix+filepath.Base(old))
		if err := fsx.RemoveAll(w.aside); err != nil {
			return swapped{}, err
		}
		if err := fsx.Rename(old, w.aside); err != nil {
			return swapped{}, err
		}
	}
	if err := fsx.Rename(tmp, w.placed); err != nil {
		if w.aside != "" {
			err = errors.Join(err, fsx.Rename(w.aside, old))
		}
		return swapped{}, err
	}
	return w, nil
}

// carryOver applies the three-way rule to every file of prof (the profile's mod folder), against old (the current
// version in the store) and target (the new copy being built).
func carryOver(prof, old, target string) error {
	return carryOverWalk(prof, old, target, false, nil)
}

// carryOverWalk applies the three-way rule. A file the old version shipped unchanged that target no longer has is
// dropped, or handed to gone when it is set.
func carryOverWalk(prof, old, target string, configOnly bool, gone func(rel, p string) error) error {
	return filepath.WalkDir(prof, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(prof, p)
		if err != nil {
			return err
		}
		if configOnly && !isUserWritten(rel) {
			return nil
		}
		oldP, newP := filepath.Join(old, rel), filepath.Join(target, rel)
		inOld := exists(oldP)
		if inOld {
			if same, err := sameFile(p, oldP); err != nil || same {
				if err == nil && gone != nil && !exists(newP) {
					return gone(rel, p)
				}
				return err
			}
		}
		if !exists(newP) {
			return copyOver(p, newP)
		}
		if inOld {
			if unchanged, err := sameFile(newP, oldP); err != nil {
				return err
			} else if unchanged {
				return copyOver(p, newP)
			}
		}
		if same, err := sameFile(p, newP); err != nil || same {
			return err
		}
		return copyOver(p, newP+conflictSuffix)
	})
}

func sameFile(a, b string) (bool, error) {
	ia, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	ib, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if ia.Size() != ib.Size() {
		return false, nil
	}
	ba, err := fsx.ReadFile(a)
	if err != nil {
		return false, err
	}
	bb, err := fsx.ReadFile(b)
	return bytes.Equal(ba, bb), err
}

func copyOver(src, dst string) error {
	b, err := fsx.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	if err := os.Remove(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return fsx.WriteFile(dst, b, 0o600)
}

func deleteOldVersion(found []manifest.Mod, uniqueID string) bool {
	for _, m := range found {
		if SameID(m.UniqueID, uniqueID) {
			return m.DeleteOldVersion
		}
	}
	return false
}

// saveBackup zips the game's Saves folder into <datadir>/backups. Games without a Saves folder need none.
func (s *Store) saveBackup(game, profileID string) error {
	if game != "stardew" {
		return nil
	}
	selected := ""
	var err error
	if s.settings != nil {
		_, selected, _, err = gamepkg.Resolve(s.home, s.settings.Get(), game)
		if err != nil {
			return err
		}
	}
	savesDir, err := gamepkg.SavesDir(selected, s.home)
	if err != nil {
		return err
	}
	set := settings.Defaults()
	if s.settings != nil {
		set = s.settings.Get()
	}
	var overrides map[string]string
	if p, err := s.read(game, profileID); err == nil {
		overrides = p.PrefOverrides()
	}
	target, err := backup.TargetFor(set, game, overrides)
	if err != nil {
		return err
	}
	_, err = backup.Saves(savesDir, target.Dir, target.Keep, time.Now(), backup.Cause{Profile: profileID, Kind: backup.KindUpdate})
	return err
}
