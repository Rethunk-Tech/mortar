package profile

import (
	"errors"
	"slices"
)

// OffersUpdate is whether this entry should show newer as an available update.
func (e Entry) OffersUpdate(newer string) bool {
	if e.Pinned || newer == "" {
		return false
	}
	return e.SkipVersion == "" || e.SkipVersion != newer
}

func (e Entry) SkipsSource(source string) bool {
	return slices.Contains(e.SkipSources, source)
}

func (s *Store) patchEntry(game, id, key string, fn func(*Entry) error) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		i := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == key })
		if i < 0 {
			return errors.New("no such mod in this profile")
		}
		return fn(&p.Entries[i])
	})
}

// SetPinned records whether the entry stays on its current version. Pinning clears a skipped update.
func (s *Store) SetPinned(game, id, key string, pinned bool) (Profile, error) {
	return s.patchEntry(game, id, key, func(e *Entry) error {
		e.Pinned = pinned
		if pinned {
			e.SkipVersion = ""
		}
		return nil
	})
}

// SetSkipVersion hides that exact newer version, or clears the skip when version is empty.
func (s *Store) SetSkipVersion(game, id, key, version string) (Profile, error) {
	return s.patchEntry(game, id, key, func(e *Entry) error {
		e.SkipVersion = version
		return nil
	})
}

// SetSkipSource records whether updates from source are hidden for the entry.
func (s *Store) SetSkipSource(game, id, key, source string, skip bool) (Profile, error) {
	return s.patchEntry(game, id, key, func(e *Entry) error {
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
