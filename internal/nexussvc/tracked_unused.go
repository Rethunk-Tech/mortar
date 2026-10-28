package nexussvc

import (
	"context"
	"errors"
)

// TrackedUnusedCount is how many of a game's tracked mods no profile uses, which UntrackAll with onlyNotInProfiles
// untracks.
func (s *Service) TrackedUnusedCount(ctx context.Context, gameID string) (int, error) {
	ids, err := s.trackedFor(ctx, gameID)
	if errors.Is(err, ErrSignedOut) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	used, err := s.profileModIDs(gameID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, id := range ids {
		if !used[id] {
			n++
		}
	}
	return n, nil
}
