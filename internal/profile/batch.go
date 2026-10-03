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
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		seen := make(map[string]bool, len(keys))
		for _, key := range keys {
			if seen[key] {
				continue
			}
			seen[key] = true
			found := false
			for i := range p.Entries {
				if p.Entries[i].Key == key {
					if err := fn(&p.Entries[i], key); err != nil {
						return err
					}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unknown profile entry %q", key)
			}
		}
		return nil
	})
}

// SetPinnedMany changes the pinned state of each named entry in one profile write.
func (s *Store) SetPinnedMany(game, id string, keys []string, pinned bool) (Profile, error) {
	return s.updateEntries(game, id, keys, func(e *Entry, _ string) error {
		e.Pinned = pinned
		if pinned {
			e.SkipVersion = ""
		}
		return nil
	})
}

// SetEntryCategoryMany changes the category override of each named entry in one profile write.
func (s *Store) SetEntryCategoryMany(game, id string, keys []string, override string) (Profile, error) {
	override = strings.TrimSpace(override)
	if override != "" {
		if idPattern.MatchString(override) {
			cats, err := readCategories(game)
			if err != nil {
				return Profile{}, err
			}
			found := false
			for _, category := range cats {
				if category.ID == override {
					found = true
					break
				}
			}
			if !found {
				return Profile{}, fmt.Errorf("unknown custom category %q", override)
			}
		} else if err := cleanCategoryName(override); err != nil {
			return Profile{}, err
		}
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
			filtered := e.Tags[:0]
			for _, existing := range e.Tags {
				if existing != tag {
					filtered = append(filtered, existing)
				}
			}
			e.Tags = filtered
		}
		_, e.Tags, err = CleanEntryNoteTags(e.Note, e.Tags)
		return err
	})
}

// SetSkipVersionMany records the skipped latest version for each entry in one profile write.
func (s *Store) SetSkipVersionMany(game, id string, refs []SkipVersionRef) (Profile, error) {
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		for _, ref := range refs {
			found := false
			for i := range p.Entries {
				if p.Entries[i].Key == ref.Key {
					p.Entries[i].SkipVersion = ref.Version
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unknown profile entry %q", ref.Key)
			}
		}
		return nil
	})
}

// EntryFields is the pin, skip, tag and category state of one profile entry.
type EntryFields struct {
	Key              string   `json:"key"`
	Pinned           bool     `json:"pinned"`
	SkipVersion      string   `json:"skipVersion"`
	Tags             []string `json:"tags"`
	CategoryOverride string   `json:"categoryOverride"`
}

// RestoreEntryFields writes each entry's previous pin, skip, tags and category in one profile write.
func (s *Store) RestoreEntryFields(game, id string, fields []EntryFields) (Profile, error) {
	if len(fields) == 0 {
		p, err := s.read(game, id)
		return p, err
	}
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		for _, field := range fields {
			found := false
			for i := range p.Entries {
				if p.Entries[i].Key != field.Key {
					continue
				}
				p.Entries[i].Pinned = field.Pinned
				p.Entries[i].SkipVersion = field.SkipVersion
				_, tags, err := CleanEntryNoteTags(p.Entries[i].Note, field.Tags)
				if err != nil {
					return err
				}
				p.Entries[i].Tags = tags
				p.Entries[i].CategoryOverride = field.CategoryOverride
				found = true
				break
			}
			if !found {
				return fmt.Errorf("unknown profile entry %q", field.Key)
			}
		}
		return nil
	})
}

// SetSkipSourceMany records whether updates from source are hidden for each entry.
func (s *Store) SetSkipSourceMany(game, id string, keys []string, source string, skip bool) (Profile, error) {
	return s.updateEntries(game, id, keys, func(e *Entry, _ string) error {
		if skip {
			if !slices.Contains(e.SkipSources, source) {
				e.SkipSources = append(e.SkipSources, source)
			}
			return nil
		}
		e.SkipSources = slices.DeleteFunc(e.SkipSources, func(existing string) bool { return existing == source })
		if len(e.SkipSources) == 0 {
			e.SkipSources = nil
		}
		return nil
	})
}
