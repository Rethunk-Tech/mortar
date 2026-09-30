package profile

import (
	"fmt"
	"slices"
	"strings"
)

// DiffSide is one profile's copy of a user mod in a comparison.
type DiffSide struct {
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Enabled  bool   `json:"enabled"`
	Key      string `json:"key"`
	Source   Source `json:"source"`
}

// DiffPair is the same UniqueID in both profiles with a different version or enabled state.
type DiffPair struct {
	UniqueID string   `json:"uniqueId"`
	Name     string   `json:"name"`
	A        DiffSide `json:"a"`
	B        DiffSide `json:"b"`
}

// Diff is the user-mod comparison of two profiles of the same game, matched by UniqueID.
type Diff struct {
	OnlyA   []DiffSide `json:"onlyA"`
	OnlyB   []DiffSide `json:"onlyB"`
	Changed []DiffPair `json:"changed"`
}

func enabledOf(e Entry, uniqueID string) bool {
	return !slices.ContainsFunc(e.Disabled, func(id string) bool { return sameID(id, uniqueID) })
}

func indexUserMods(p Profile) map[string]DiffSide {
	out := map[string]DiffSide{}
	for _, e := range p.Entries {
		if isBundled(e) {
			continue
		}
		for _, m := range e.Mods {
			k := strings.ToLower(m.UniqueID)
			if _, ok := out[k]; ok {
				continue
			}
			out[k] = DiffSide{
				UniqueID: m.UniqueID, Name: m.Name, Version: m.Version, Enabled: enabledOf(e, m.UniqueID),
				Key: e.Key, Source: e.Source,
			}
		}
	}
	return out
}

func sortSides(sides []DiffSide) {
	slices.SortFunc(sides, func(a, b DiffSide) int {
		if n := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); n != 0 {
			return n
		}
		return strings.Compare(strings.ToLower(a.UniqueID), strings.ToLower(b.UniqueID))
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
			d.Changed = append(d.Changed, DiffPair{UniqueID: side.UniqueID, Name: name, A: side, B: other})
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
		if n := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); n != 0 {
			return n
		}
		return strings.Compare(strings.ToLower(a.UniqueID), strings.ToLower(b.UniqueID))
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

// Diff compares two profiles of the same game.
func (s *Store) Diff(game, aID, bID string) (Diff, error) {
	a, err := s.load(game, aID)
	if err != nil {
		return Diff{}, err
	}
	b, err := s.load(game, bID)
	if err != nil {
		return Diff{}, err
	}
	return DiffProfiles(a, b), nil
}

func destHasKey(p Profile, key string) bool {
	return slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == key })
}

// CopyMods copies the named user mods from one profile into another by their store keys, with no download.
// A running game locks the destination the same way AddEntry does.
func (s *Store) CopyMods(game, fromID, toID string, uniqueIDs []string) (Profile, error) {
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
		side, ok := src[strings.ToLower(id)]
		if !ok {
			return Profile{}, fmt.Errorf("no mod %q in this profile", id)
		}
		to, err := s.load(game, toID)
		if err != nil {
			return Profile{}, err
		}
		if destHasKey(to, side.Key) {
			last, err = s.SetModEnabled(game, toID, side.Key, side.UniqueID, side.Enabled)
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
			last, err = s.SetModEnabled(game, toID, side.Key, side.UniqueID, false)
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
