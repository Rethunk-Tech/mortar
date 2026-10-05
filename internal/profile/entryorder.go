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
