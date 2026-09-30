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
	"github.com/Rethunk-AI/mortar/internal/manifest"
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

// RollBack swaps the entry key back to its previous version, carrying over what the user and the mod wrote.
func (s *Store) RollBack(game, id, key string) (Profile, error) {
	return s.moveTo(game, id, key, "", nil)
}

// moveTo switches an entry to another store item. An empty newKey means the entry's previous key. A source
// replaces the entry's own.
func (s *Store) moveTo(game, id, oldKey, newKey string, source *Source) (Profile, error) {
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.updateLocked(game, id, func(p *Profile, dir string) error {
		ei := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == oldKey })
		if ei < 0 {
			return fmt.Errorf("%q is not in this profile", oldKey)
		}
		if newKey == "" {
			if newKey = p.Entries[ei].PreviousKey; newKey == "" {
				return fmt.Errorf("%q has no previous version to roll back to", oldKey)
			}
		}
		for _, e := range p.Entries {
			if e.Key == newKey {
				return &DuplicateError{Key: newKey, Label: entryLabel(e)}
			}
		}
		ne, err := s.swapEntry(game, dir, p.Entries[ei], newKey)
		if err != nil {
			return err
		}
		if source != nil {
			ne.Source = *source
		}
		p.Entries[ei] = ne
		return nil
	})
	if err != nil {
		return Profile{}, err
	}
	return p, s.items.Touch(game, oldKey, newKey)
}

// swapEntry builds the target version's folder beside mods/, carries over the profile's files, and renames it
// over the old one. It returns the entry as it stands afterwards.
func (s *Store) swapEntry(game, dir string, e Entry, newKey string) (Entry, error) {
	oldSrc, err := s.items.Path(game, e.Key)
	if err != nil {
		return Entry{}, err
	}
	newSrc, err := s.items.Path(game, newKey)
	if err != nil {
		return Entry{}, err
	}
	found, err := manifest.Scan(newSrc)
	if err != nil {
		return Entry{}, err
	}
	if len(found) == 0 {
		return Entry{}, &NoModError{Key: newKey}
	}
	ne := Entry{Key: newKey, PreviousKey: e.Key, Source: e.Source, Mods: entryMods(found), Disabled: []string{}}
	for _, m := range ne.Mods {
		if hasID(e.Disabled, m.UniqueID) {
			ne.Disabled = append(ne.Disabled, m.UniqueID)
		}
	}
	if err := s.saveBackup(game); err != nil {
		return Entry{}, fmt.Errorf("back up saves: %w", err)
	}
	modsDir := filepath.Join(dir, "mods")
	if err := os.MkdirAll(modsDir, 0o700); err != nil {
		return Entry{}, err
	}
	tmp, err := os.MkdirTemp(modsDir, tempPrefix)
	if err != nil {
		return Entry{}, err
	}
	err = fillUpdate(tmp, modsDir, oldSrc, newSrc, e, ne)
	if err != nil {
		return Entry{}, errors.Join(err, os.RemoveAll(tmp))
	}
	return ne, nil
}

func fillUpdate(tmp, modsDir, oldSrc, newSrc string, e, ne Entry) error {
	if err := datadir.CopyTree(newSrc, tmp); err != nil {
		return err
	}
	for _, nm := range ne.Mods {
		i := slices.IndexFunc(e.Mods, func(m EntryMod) bool { return sameID(m.UniqueID, nm.UniqueID) })
		if i < 0 {
			continue
		}
		plain, dotted, err := pair(modsDir, e.Key, e.Mods[i].Folder)
		if err != nil {
			return err
		}
		cur := plain
		if !exists(cur) {
			cur = dotted
		}
		if !exists(cur) {
			continue
		}
		err = carryOver(cur, filepath.Join(oldSrc, filepath.FromSlash(e.Mods[i].Folder)), filepath.Join(tmp, filepath.FromSlash(nm.Folder)))
		if err != nil {
			return err
		}
	}
	final, err := materialize(tmp, ne)
	if err != nil {
		return err
	}
	return replaceFolder(modsDir, e.Key, tmp, final)
}

// replaceFolder moves the old entry folder aside, renames tmp into place, and restores the old folder if that fails.
func replaceFolder(modsDir, oldKey, tmp, final string) error {
	old := filepath.Join(modsDir, oldKey)
	if !exists(old) {
		old = filepath.Join(modsDir, "."+oldKey)
	}
	aside := tmp + ".old"
	hadOld := exists(old)
	if hadOld {
		if err := os.Rename(old, aside); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, filepath.Join(modsDir, final)); err != nil {
		if hadOld {
			err = errors.Join(err, os.Rename(aside, old))
		}
		return err
	}
	// A failed delete leaves a temp-prefixed folder, which the next rebuild sweeps.
	_ = os.RemoveAll(aside)
	return nil
}

// carryOver applies the three-way rule to every file of prof (the profile's mod folder), against old (the current
// version in the store) and target (the new copy being built).
func carryOver(prof, old, target string) error {
	return filepath.WalkDir(prof, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(prof, p)
		if err != nil {
			return err
		}
		oldP, newP := filepath.Join(old, rel), filepath.Join(target, rel)
		inOld := exists(oldP)
		if inOld {
			if same, err := sameFile(p, oldP); err != nil || same {
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
	return fsx.WriteFile(dst, b, 0o600)
}

// saveBackup zips the game's Saves folder into <datadir>/backups. Games without a Saves folder need none.
func (s *Store) saveBackup(game string) error {
	if game != "stardew" {
		return nil
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	_, err = backup.Saves(filepath.Join(cfg, "StardewValley", "Saves"), filepath.Join(filepath.Dir(s.root), "backups"), time.Now())
	return err
}
