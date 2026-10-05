package profile

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// DiffSide is one profile's copy of a user mod in a comparison.
type DiffSide struct {
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
	Key     string `json:"key"`
	Source  Source `json:"source"`
}

// DiffPair is the same mod id in both profiles, with each profile's copy.
type DiffPair struct {
	ID   mod.ID   `json:"id"`
	Name string   `json:"name"`
	A    DiffSide `json:"a"`
	B    DiffSide `json:"b"`
}

func indexUserMods(p Profile) map[string]DiffSide {
	out := map[string]DiffSide{}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		for _, m := range e.Mods {
			k := m.ID.Fold()
			if _, ok := out[k]; ok {
				continue
			}
			out[k] = DiffSide{
				ID: m.ID, Name: m.Name, Version: m.Version, Enabled: e.Enabled(m.ID),
				Key: e.Key, Source: e.Source,
			}
		}
	}
	return out
}

// compareNameThenID orders mods by name, then mod id, ignoring case.
func compareNameThenID(aName string, aID mod.ID, bName string, bID mod.ID) int {
	return cmp.Or(
		strings.Compare(strings.ToLower(aName), strings.ToLower(bName)),
		strings.Compare(aID.Fold(), bID.Fold()),
	)
}

func sortSides(sides []DiffSide) {
	slices.SortFunc(sides, func(a, b DiffSide) int {
		return compareNameThenID(a.Name, a.ID, b.Name, b.ID)
	})
}

func (s *Store) load(game, id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(game, id)
}

func destHasKey(p Profile, key string) bool {
	return slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == key })
}

// CopyMods copies the named user mods from one profile into another by their store keys, with no download.
// A running game locks the destination the same way AddEntry does.
func (s *Store) CopyMods(game, fromID, toID string, uniqueIDs []mod.ID) (Profile, error) {
	if fromID == toID {
		return Profile{}, fmt.Errorf("pick a different profile to copy into")
	}
	from, err := s.load(game, fromID)
	if err != nil {
		return Profile{}, err
	}
	src := indexUserMods(from)
	var last Profile
	for _, id := range uniqueIDs {
		side, ok := src[id.Fold()]
		if !ok {
			return Profile{}, fmt.Errorf("no mod %q in this profile", id)
		}
		to, err := s.load(game, toID)
		if err != nil {
			return Profile{}, err
		}
		if destHasKey(to, side.Key) {
			last, err = s.SetModEnabled(game, toID, side.Key, side.ID, side.Enabled)
			if err != nil {
				return Profile{}, err
			}
			continue
		}
		last, err = s.AddEntry(game, toID, side.Key, side.Source)
		if err != nil {
			return Profile{}, err
		}
		if !side.Enabled {
			last, err = s.SetModEnabled(game, toID, side.Key, side.ID, false)
			if err != nil {
				return Profile{}, err
			}
		}
	}
	if last.ID == "" {
		return s.load(game, toID)
	}
	return last, nil
}
