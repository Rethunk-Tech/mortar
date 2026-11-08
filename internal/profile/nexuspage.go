package profile

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// Unsourced is whether the source names no page to get the entry from again: a file the user picked, or a Nexus
// entry with no page id.
func (s Source) Unsourced() bool {
	return s.Kind == KindLocal || (s.Kind == KindNexus && s.ModID <= 0)
}

// withOwnFile is the source of an entry whose files the player replaced with an archive of their own: the same page,
// picture and names, so its links, details and update checks carry on, without what named the site's file.
func (s Source) withOwnFile(mods []Component) Source {
	s.FileID, s.Digest, s.Tag, s.Asset, s.Version = 0, "", "", "", ""
	if len(mods) > 0 {
		s.Version = mods[0].Version
	}
	return s
}

// NexusFile names a Nexus page and, where known, the file on it; FileID is 0 for an unknown file.
type NexusFile struct {
	ModID, FileID int
}

// SetNexusFiles records the Nexus page of each named entry that has no source of its own, which gives it that
// page's links and update checks, and the file of each that is on that page without one, which lets a share carry
// it. found maps entry keys to what was found.
func (s *Store) SetNexusFiles(game, id string, found map[string]NexusFile) (Profile, error) {
	note := HistoryEvent{Change: ChangeSourced}
	return s.updateEntriesAs(game, id, historyBulk, note, slices.Sorted(maps.Keys(found)), func(e *Entry, key string) error {
		f := found[key]
		if e.Source.Unsourced() {
			e.Source.Kind, e.Source.ModID = KindNexus, f.ModID
			if len(e.Mods) > 0 {
				e.Source.Version = e.Mods[0].Version
			}
		}
		if e.Source.Kind == KindNexus && e.Source.ModID == f.ModID && e.Source.FileID == 0 {
			e.Source.FileID = f.FileID
		}
		return nil
	})
}

// SetNexusFile records the Nexus page, and the file on it when known, that the player named for one entry. It
// replaces a page a lookup got wrong; an entry from another site keeps its source.
func (s *Store) SetNexusFile(game, id, key string, f NexusFile) (Profile, error) {
	if f.ModID <= 0 || f.FileID < 0 {
		return Profile{}, errors.New("a Nexus page has a positive id")
	}
	p, err := s.read(game, id)
	if err != nil {
		return Profile{}, err
	}
	i, err := requireEntry(p.Entries, key)
	if err != nil {
		return Profile{}, err
	}
	note := HistoryEvent{Change: ChangeSourced, Name: p.Entries[i].Label()}
	return s.updateEntriesAs(game, id, historyBulk, note, []string{key}, func(e *Entry, _ string) error {
		if e.Source.Kind != KindLocal && e.Source.Kind != KindNexus {
			return fmt.Errorf("%s comes from another site", e.Label())
		}
		e.Source.Kind, e.Source.ModID, e.Source.FileID, e.Source.Digest = KindNexus, f.ModID, f.FileID, ""
		if len(e.Mods) > 0 {
			e.Source.Version = e.Mods[0].Version
		}
		return nil
	})
}
