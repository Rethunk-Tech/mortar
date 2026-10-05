package control

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (s *Services) applyEverywhere(ctx context.Context, p Params) (profile.EverywhereResult, error) {
	key := p.Key
	if key == "" {
		key = "latest"
	}
	if len(p.IDs) > 0 {
		return s.Profiles.UpdateEverywhere(p.Game, p.IDs[0], key)
	}
	profiles, err := s.Profiles.List(p.Game)
	if err != nil {
		return profile.EverywhereResult{}, err
	}
	seen := map[string]struct{}{}
	var out profile.EverywhereResult
	for _, prof := range profiles {
		r, err := s.Problems.Updates(ctx, p.Game, prof.ID)
		if err != nil {
			return profile.EverywhereResult{}, err
		}
		for _, u := range r.Updates {
			id := u.ID.Fold()
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			part, err := s.Profiles.UpdateEverywhere(p.Game, string(u.ID), key)
			if err != nil {
				return out, err
			}
			out.Updated = append(out.Updated, part.Updated...)
			out.Skipped = append(out.Skipped, part.Skipped...)
		}
	}
	return out, nil
}
