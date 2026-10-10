package profile

import (
	"fmt"
	"slices"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// fileKeySep joins a store item's key and a file's path into the key of a per-file entry; no store key holds it.
const fileKeySep = "#"

// sameArchiveLine reports whether b is another file of the page, repository or local file a was installed from, so
// an install of b replaces a. A file's own id or tag is what tells the versions apart; nothing is fingerprinted.
func sameArchiveLine(a, b Source) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch {
	case a.ModID != 0 || b.ModID != 0:
		return a.ModID == b.ModID
	case a.Repo != "" || b.Repo != "":
		return a.Repo == b.Repo
	}
	return a.Name != "" && a.Name == b.Name
}

// folderEntries are the entries a folder-loader game's store item key becomes: one per laid-out file, or the one
// entry holding them all when a file in it only works beside its siblings. Each keeps the archive's source.
func (s *Store) folderEntries(game, id, key string, source Source, whole []Component) ([]Entry, error) {
	_, l, _, err := s.layoutOf(game, id, key, nil)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if !installerGame(game).Splits(l) {
		return []Entry{{Key: key, Source: source, Mods: whole, Disabled: []mod.ID{}, Added: now, Package: true}}, nil
	}
	out := make([]Entry, 0, len(l.Files))
	for _, f := range l.Files {
		k := key + fileKeySep + f.Rel
		out = append(out, Entry{
			Key: k, Item: key, File: f.Rel, Source: source, Disabled: []mod.ID{}, Added: now, Package: true,
			Mods: []Component{{ID: mod.NewID(mod.FormatFolder, k), Name: f.Rel, Folder: "."}},
		})
	}
	return out, nil
}

// placeFolderLocked adds a folder-loader game's store item to the profile as folderEntries says. The entries of
// an earlier file of the same page or repository are all replaced together, each file keeping its on or off state.
func (s *Store) placeFolderLocked(game, id, key string, source Source, whole []Component) (Profile, bool, bool, error) {
	fresh, err := s.folderEntries(game, id, key, source, whole)
	if err != nil {
		return Profile{}, false, false, err
	}
	var updated bool
	p, err := s.updateLocked(game, id, func(p *Profile, _ string) error {
		for _, e := range p.Entries {
			if slices.ContainsFunc(fresh, func(n Entry) bool { return n.Key == e.Key }) {
				return &DuplicateError{Key: e.Key, Label: entryLabel(e)}
			}
		}
		off := map[string]bool{}
		p.Entries = slices.DeleteFunc(p.Entries, func(e Entry) bool {
			if e.IsOverlay() || !e.Package || !sameArchiveLine(e.Source, source) || e.StoreKey() == key {
				return false
			}
			off[e.File] = !e.hasPackageEnabled()
			return true
		})
		if updated = len(off) > 0; updated {
			if err := s.saveBackup(game, id); err != nil {
				return fmt.Errorf("back up saves: %w", err)
			}
		}
		for i := range fresh {
			if off[fresh[i].File] {
				for _, m := range fresh[i].Mods {
					fresh[i].Disabled = append(fresh[i].Disabled, m.ID)
				}
			}
		}
		p.Entries = append(p.Entries, fresh...)
		return nil
	})
	if err != nil {
		return Profile{}, false, false, err
	}
	return p, updated, updated, s.items.Touch(game, key)
}
