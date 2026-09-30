package profile

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// MaxEntryNote caps an entry's note, in characters.
	MaxEntryNote = 500
	// MaxEntryTags is how many tags an entry may hold.
	MaxEntryTags = 8
	// MaxEntryTag caps one tag, in characters.
	MaxEntryTag = 24
)

// CleanEntryNoteTags trims a note and tags and checks the caps.
func CleanEntryNoteTags(note string, tags []string) (string, []string, error) {
	note = strings.TrimSpace(note)
	if n := utf8.RuneCountInString(note); n > MaxEntryNote {
		return "", nil, fmt.Errorf("note is longer than %d characters", MaxEntryNote)
	}
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > MaxEntryTag {
			return "", nil, fmt.Errorf("tag is longer than %d characters", MaxEntryTag)
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	if len(out) > MaxEntryTags {
		return "", nil, fmt.Errorf("more than %d tags", MaxEntryTags)
	}
	if len(out) == 0 {
		out = nil
	}
	return note, out, nil
}

// SetEntryNoteTags records the note and tags on one profile entry.
func (s *Store) SetEntryNoteTags(game, id, key, note string, tags []string) (Profile, error) {
	note, tags, err := CleanEntryNoteTags(note, tags)
	if err != nil {
		return Profile{}, err
	}
	return s.patchEntry(game, id, key, func(e *Entry) error {
		e.Note = note
		e.Tags = tags
		return nil
	})
}
