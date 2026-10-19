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
		i := entryIndex(p.Entries, key)
		if i < 0 {
			return errors.New("no such mod in this profile")
		}
		return fn(&p.Entries[i])
	})
}

// SetPinned records whether the entry stays on its current version. Pinning clears a skipped update.
func (s *Store) SetPinned(game, id, key string, pinned bool, pinReason string) (Profile, error) {
	reason, err := cleanPinReason(pinReason)
	if err != nil {
		return Profile{}, err
	}
	return s.patchEntry(game, id, key, func(e *Entry) error {
		e.Pinned = pinned
		if pinned {
			e.SkipVersion = ""
			e.PinReason = reason
		} else {
			e.PinReason = ""
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
		setSkipSource(e, source, skip)
		return nil
	})
}

func setSkipSource(e *Entry, source string, skip bool) {
	if skip {
		if !e.SkipsSource(source) {
			e.SkipSources = append(e.SkipSources, source)
		}
		return
	}
	e.SkipSources = slices.DeleteFunc(e.SkipSources, func(existing string) bool { return existing == source })
	if len(e.SkipSources) == 0 {
		e.SkipSources = nil
	}
}

// FrozenPinReason marks the pins a freeze made, so Unfreeze lifts those and keeps the ones the player set.
const FrozenPinReason = "Frozen with the profile"

// frozen reports whether a freeze holds the profile: every mod is pinned and a freeze made at least one pin. A mod
// added to such a profile is held too, so it is not offered updates the rest are spared.
func frozen(entries []Entry) bool {
	held, made := 0, false
	for _, e := range entries {
		if e.OverlayOf != "" {
			continue
		}
		if !e.Pinned {
			return false
		}
		held++
		made = made || e.PinReason == FrozenPinReason
	}
	return held > 0 && made
}

// SetFrozen holds every mod of the profile at its current version, or lifts the pins a freeze made. A mod the player
// pinned already keeps their pin and reason either way. Pinned mods are offered no update.
func (s *Store) SetFrozen(game, id string, frozen bool) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		for i := range p.Entries {
			e := &p.Entries[i]
			switch {
			case frozen && !e.Pinned && e.OverlayOf == "":
				e.Pinned, e.SkipVersion, e.PinReason = true, "", FrozenPinReason
			case !frozen && e.Pinned && e.PinReason == FrozenPinReason:
				e.Pinned, e.PinReason = false, ""
			}
		}
		return nil
	})
}
