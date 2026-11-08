package profile

import (
	"fmt"
	"slices"
	"strings"
)

// SkipVersionRef identifies the latest version to skip for one profile entry.
type SkipVersionRef struct {
	Key     string `json:"key"`
	Version string `json:"version"`
}

func (s *Store) updateEntries(game, id string, keys []string, fn func(*Entry, string) error) (Profile, error) {
	return s.updateEntriesAs(game, id, "", HistoryEvent{}, keys, fn)
}

// updateEntriesAs is updateEntries with the history event the change is recorded as; an empty kind lets the
// change name itself.
func (s *Store) updateEntriesAs(game, id, kind string, note HistoryEvent, keys []string, fn func(*Entry, string) error) (Profile, error) {
	if err := s.lockUnlocked(game, id); err != nil {
		return Profile{}, err
	}
	defer s.mu.Unlock()
	return s.updateLockedAs(game, id, kind, note, func(p *Profile, _ string) error {
		seen := make(map[string]bool, len(keys))
		for _, key := range keys {
			if seen[key] {
				continue
			}
			seen[key] = true
			i, err := requireEntry(p.Entries, key)
			if err != nil {
				return err
			}
			if err := fn(&p.Entries[i], key); err != nil {
				return err
			}
		}
		return nil
	})
}

// applyPin records whether an entry stays on its current version: pinning clears a skipped update and sets the reason,
// unpinning drops the reason.
func applyPin(e *Entry, pinned bool, reason string) {
	e.Pinned = pinned
	if pinned {
		e.SkipVersion = ""
		e.PinReason = reason
	} else {
		e.PinReason = ""
	}
}

// SetPinnedMany changes the pinned state of each named entry in one profile write.
func (s *Store) SetPinnedMany(game, id string, keys []string, pinned bool, pinReason string) (Profile, error) {
	reason, err := cleanPinReason(pinReason)
	if err != nil {
		return Profile{}, err
	}
	return s.updateEntries(game, id, keys, func(e *Entry, _ string) error {
		applyPin(e, pinned, reason)
		return nil
	})
}

// SetEntryCategoryMany changes the category override of each named entry in one profile write.
func (s *Store) SetEntryCategoryMany(game, id string, keys []string, override string) (Profile, error) {
	override, err := s.categoryOverride(game, override)
	if err != nil {
		return Profile{}, err
	}
	return s.updateEntries(game, id, keys, func(e *Entry, _ string) error {
		e.CategoryOverride = override
		return nil
	})
}

// SetEntryTagsMany adds or removes one tag on each named entry in one profile write.
func (s *Store) SetEntryTagsMany(game, id string, keys []string, tag string, add bool) (Profile, error) {
	tag = strings.TrimSpace(tag)
	_, cleaned, err := CleanEntryNoteTags("", []string{tag})
	if err != nil {
		return Profile{}, err
	}
	if len(cleaned) != 1 {
		return Profile{}, fmt.Errorf("tag must not be empty")
	}
	tag = cleaned[0]
	return s.updateEntries(game, id, keys, func(e *Entry, _ string) error {
		if add {
			if slices.Contains(e.Tags, tag) {
				return nil
			}
			e.Tags = append(e.Tags, tag)
		} else {
			e.Tags = slices.DeleteFunc(e.Tags, func(existing string) bool { return existing == tag })
		}
		_, e.Tags, err = CleanEntryNoteTags(e.Note, e.Tags)
		return err
	})
}

// SetSkipVersionMany records the skipped latest version for each entry in one profile write.
func (s *Store) SetSkipVersionMany(game, id string, refs []SkipVersionRef) (Profile, error) {
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		for _, ref := range refs {
			i, err := requireEntry(p.Entries, ref.Key)
			if err != nil {
				return err
			}
			p.Entries[i].SkipVersion = ref.Version
		}
		return nil
	})
}

// EntryFields is the pin, skip, tag, category and install reason state of one profile entry.
type EntryFields struct {
	Key              string   `json:"key"`
	Pinned           bool     `json:"pinned"`
	SkipVersion      string   `json:"skipVersion"`
	Tags             []string `json:"tags"`
	CategoryOverride string   `json:"categoryOverride"`
	Implicit         bool     `json:"implicit"`
}

// RestoreEntryFields writes each entry's previous pin, skip, tags, category and install reason in one profile write.
func (s *Store) RestoreEntryFields(game, id string, fields []EntryFields) (Profile, error) {
	if len(fields) == 0 {
		p, err := s.read(game, id)
		return p, err
	}
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		for _, field := range fields {
			i, err := requireEntry(p.Entries, field.Key)
			if err != nil {
				return err
			}
			e := &p.Entries[i]
			e.Pinned = field.Pinned
			e.SkipVersion = field.SkipVersion
			_, tags, err := CleanEntryNoteTags(e.Note, field.Tags)
			if err != nil {
				return err
			}
			e.Tags = tags
			e.CategoryOverride = field.CategoryOverride
			if !e.Source.Bundled() {
				e.Implicit = field.Implicit
			}
		}
		return nil
	})
}

// SetSkipSourceMany records whether updates from source are hidden for each entry.
func (s *Store) SetSkipSourceMany(game, id string, keys []string, source string, skip bool) (Profile, error) {
	return s.updateEntries(game, id, keys, func(e *Entry, _ string) error {
		setSkipSource(e, source, skip)
		return nil
	})
}
