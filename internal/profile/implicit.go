package profile

import (
	"errors"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

type intent int8

const (
	intentExplicit intent = iota
	intentImplicit
	// intentLink is a download the player started from a Nexus link, which Nexus also makes them use for a
	// dependency; newImplicit decides it from the profile.
	intentLink
)

// WithImplicit returns a copy that creates its entry implicit (brought in for another mod) or explicit.
func (s Source) WithImplicit(implicit bool) Source {
	s.intent = intentExplicit
	if implicit {
		s.intent = intentImplicit
	}
	return s
}

// WithFromLink returns a copy whose entry is implicit when it supplies a mod the profile requires and lacks.
func (s Source) WithFromLink() Source {
	s.intent = intentLink
	return s
}

// startsImplicit is whether a new entry from this source is implicit without looking at the profile: Mortar's own
// loader and bridge entries always are.
func (s Source) startsImplicit() bool {
	return s.Bundled() || s.intent == intentImplicit
}

// newImplicit is whether a new entry holding mods starts implicit, judged on the profile's entries before it is added.
func newImplicit(entries []Entry, source Source, mods []Component) bool {
	return source.startsImplicit() || (source.intent == intentLink && suppliesMissing(entries, mods))
}

// suppliesMissing is whether mods hold an id that an enabled mod in entries requires (a required dependency or the
// framework of a content pack) and no entry provides.
func suppliesMissing(entries []Entry, mods []Component) bool {
	var have []mod.ID
	haveVS := map[string]bool{}
	for _, e := range entries {
		for _, m := range e.Mods {
			have = append(have, m.ID)
			haveVS[m.ModID] = true
		}
	}
	fills := func(id mod.ID) bool {
		return id != "" && !hasID(have, id) && slices.ContainsFunc(mods, func(m Component) bool { return mod.Equal(m.ID, id) })
	}
	fillsVS := func(id string) bool {
		return !haveVS[id] && slices.ContainsFunc(mods, func(m Component) bool { return m.ModID != "" && m.ModID == id })
	}
	for _, e := range entries {
		for _, m := range e.Mods {
			if hasID(e.Disabled, m.ID) {
				continue
			}
			if fills(m.ContentPackFor) || slices.ContainsFunc(m.Requires, fillsVS) {
				return true
			}
			for _, need := range m.Needs {
				if !hasID(m.Optional, need) && fills(need) {
					return true
				}
			}
		}
	}
	return false
}

// SetImplicitMany marks each named entry implicit or explicit in one profile write and one history event. Mortar's
// own entries are always implicit and are refused.
func (s *Store) SetImplicitMany(game, id string, keys []string, implicit bool) (Profile, error) {
	note := HistoryEvent{Change: ChangeMarkedExplicit, Count: len(keys)}
	if implicit {
		note.Change = ChangeMarkedImplicit
	}
	return s.updateEntriesAs(game, id, historyBulk, note, keys, func(e *Entry, _ string) error {
		if e.Source.Bundled() {
			return usererr.Wrap(usererr.Invalid, errors.New("the loader and bridge entries Mortar adds are always implicit"))
		}
		if len(keys) == 1 {
			s.historyNote.Name = e.Label()
		}
		e.Implicit = implicit
		return nil
	})
}

// InferImplicit is the install reason an imported list implies when it records none: requires maps each listed item
// to the listed items it requires, and an item is implicit when another item requires it that it does not itself
// lead back to, so the members of a cycle nothing outside requires stay explicit.
func InferImplicit(requires map[string][]string) map[string]bool {
	reaches := func(from, to string) bool {
		seen := map[string]bool{}
		stack := slices.Clone(requires[from])
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if n == to {
				return true
			}
			if !seen[n] {
				seen[n] = true
				stack = append(stack, requires[n]...)
			}
		}
		return false
	}
	out := map[string]bool{}
	for by, needs := range requires {
		for _, item := range needs {
			if item != by && !reaches(item, by) {
				out[item] = true
			}
		}
	}
	return out
}

// SetIntentOffered records that the player answered the offer to mark a profile's older libraries implicit.
func (s *Store) SetIntentOffered(game, id string) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		p.IntentOffered = true
		return nil
	})
}
