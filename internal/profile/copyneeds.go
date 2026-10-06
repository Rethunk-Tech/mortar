package profile

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// NeedsToCopy is what the mods ids require from the profile fromID, followed through what those require in turn,
// that the profile toID lacks at a version the requiring mod accepts. The ids themselves are left out, and so is a
// need fromID does not hold either.
func (s *Store) NeedsToCopy(game, fromID, toID string, ids []mod.ID) ([]DiffSide, error) {
	from, err := s.load(game, fromID)
	if err != nil {
		return nil, err
	}
	have, err := s.Installed(game, fromID)
	if err != nil {
		return nil, err
	}
	target, err := s.Installed(game, toID)
	if err != nil {
		return nil, err
	}
	scheme := gamereg.VersionScheme(game)
	src := indexUserMods(from)
	seen := map[string]bool{}
	for _, id := range ids {
		seen[id.Fold()] = true
	}
	out := []DiffSide{}
	for queue := slices.Clone(ids); len(queue) > 0; queue = queue[1:] {
		for _, m := range have {
			if !mod.Equal(m.ModID(), queue[0]) {
				continue
			}
			for _, d := range m.Dependencies {
				side, inSource := src[d.ModID().Fold()]
				if !d.Required || seen[d.ModID().Fold()] || !inSource {
					continue
				}
				seen[d.ModID().Fold()] = true
				if slices.ContainsFunc(target, func(x Installed) bool {
					return mod.Equal(x.ModID(), d.ModID()) && deps.Satisfies(scheme, x.Version, d.MinimumVersion)
				}) {
					continue
				}
				out = append(out, side)
				queue = append(queue, d.ModID())
			}
		}
	}
	return out, nil
}

// CopyModsWithNeeds is CopyMods of the mods ids and of NeedsToCopy for them.
func (s *Store) CopyModsWithNeeds(game, fromID, toID string, ids []mod.ID) (Profile, error) {
	needs, err := s.NeedsToCopy(game, fromID, toID, ids)
	if err != nil {
		return Profile{}, err
	}
	all := slices.Clone(ids)
	for _, n := range needs {
		all = append(all, n.ID)
	}
	return s.CopyMods(game, fromID, toID, all)
}
