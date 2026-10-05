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

// DiffPair is the same UniqueID in both profiles with a different version or enabled state.
type DiffPair struct {
	ID   mod.ID   `json:"id"`
	Name string   `json:"name"`
	A    DiffSide `json:"a"`
	B    DiffSide `json:"b"`
}

// Diff is the user-mod comparison of two profiles of the same game, matched by UniqueID.
type Diff struct {
	OnlyA   []DiffSide `json:"onlyA"`
	OnlyB   []DiffSide `json:"onlyB"`
	Changed []DiffPair `json:"changed"`
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

// compareNameThenID orders mods by name, then UniqueID, ignoring case.
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

// DiffProfiles lists user mods only in a, only in b, and in both with a different version or enabled state.
func DiffProfiles(a, b Profile) Diff {
	left, right := indexUserMods(a), indexUserMods(b)
	var d Diff
	for k, side := range left {
		other, ok := right[k]
		if !ok {
			d.OnlyA = append(d.OnlyA, side)
			continue
		}
		if side.Version != other.Version || side.Enabled != other.Enabled {
			name := side.Name
			if name == "" {
				name = other.Name
			}
			d.Changed = append(d.Changed, DiffPair{ID: side.ID, Name: name, A: side, B: other})
		}
	}
	for k, side := range right {
		if _, ok := left[k]; !ok {
			d.OnlyB = append(d.OnlyB, side)
		}
	}
	sortSides(d.OnlyA)
	sortSides(d.OnlyB)
	slices.SortFunc(d.Changed, func(a, b DiffPair) int {
		return compareNameThenID(a.Name, a.ID, b.Name, b.ID)
	})
	if d.OnlyA == nil {
		d.OnlyA = []DiffSide{}
	}
	if d.OnlyB == nil {
		d.OnlyB = []DiffSide{}
	}
	if d.Changed == nil {
		d.Changed = []DiffPair{}
	}
	return d
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
