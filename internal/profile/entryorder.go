package profile

import (
	"errors"
	"slices"
)

var errInOrder = errors.New("already in order")

// FollowOrder puts the entries rank knows in rank's order, in the places those entries hold now; every other entry
// keeps its place. A shared profile's mods arrive in download order, appended, and this puts them back in the
// order they were shared in. Overlay entries follow their base and are left where they are. A profile already in
// that order is not written.
func (s *Store) FollowOrder(game, id string, rank func(Entry) (int, bool)) error {
	_, err := s.update(game, id, func(p *Profile, _ string) error {
		type ranked struct{ at, rank int }
		var rs []ranked
		for i, e := range p.Entries {
			if r, ok := rank(e); ok && !e.IsOverlay() {
				rs = append(rs, ranked{i, r})
			}
		}
		order := slices.Clone(rs)
		slices.SortStableFunc(order, func(a, b ranked) int { return a.rank - b.rank })
		if slices.Equal(order, rs) {
			return errInOrder
		}
		was := slices.Clone(p.Entries)
		for i, r := range rs {
			p.Entries[r.at] = was[order[i].at]
		}
		return nil
	})
	if errors.Is(err, errInOrder) {
		return nil
	}
	return err
}

// PlaceArrivals moves each entry rank knows whose key is not in placed, and only those, to just after the last placed
// entry rank puts before it, or when none is present, to just before the first placed entry rank puts after it. A
// shared profile's mods arrive one download at a time; placing only the newcomer keeps the moves the player made in
// the meantime. It returns placed with the arrivals added.
func (s *Store) PlaceArrivals(game, id string, rank func(Entry) (int, bool), placed []string) ([]string, error) {
	out := slices.Clone(placed)
	_, err := s.update(game, id, func(p *Profile, _ string) error {
		out = slices.Clone(placed)
		type arrival struct {
			key  string
			rank int
		}
		var arrivals []arrival
		for _, e := range p.Entries {
			if r, ok := rank(e); ok && !e.IsOverlay() && !slices.Contains(out, e.Key) {
				arrivals = append(arrivals, arrival{e.Key, r})
			}
		}
		if len(arrivals) == 0 {
			return errInOrder
		}
		slices.SortStableFunc(arrivals, func(a, b arrival) int { return a.rank - b.rank })
		was := slices.Clone(p.Entries)
		for _, a := range arrivals {
			at := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == a.key })
			moving := p.Entries[at]
			p.Entries = slices.Delete(p.Entries, at, at+1)
			to, after, before := at, -1, -1
			for i, e := range p.Entries {
				r, ok := rank(e)
				if !ok || e.IsOverlay() || !slices.Contains(out, e.Key) {
					continue
				}
				if r < a.rank {
					after = i
				} else if r > a.rank && before < 0 {
					before = i
				}
			}
			if after >= 0 {
				to = after + 1
			} else if before >= 0 {
				to = before
			}
			p.Entries = slices.Insert(p.Entries, to, moving)
			out = append(out, a.key)
		}
		if slices.EqualFunc(was, p.Entries, func(a, b Entry) bool { return a.Key == b.Key }) {
			return errInOrder
		}
		return nil
	})
	if errors.Is(err, errInOrder) {
		return out, nil
	}
	if err != nil {
		return placed, err
	}
	return out, nil
}
