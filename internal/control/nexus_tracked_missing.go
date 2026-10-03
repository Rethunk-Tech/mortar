package control

import (
	"context"
	"errors"
)

func (s *Services) nexusTrackedMissing(ctx context.Context, p Params) (any, error) {
	if s.Nexus == nil {
		return nil, errors.New("nexus is unavailable")
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	return s.Nexus.TrackedMissing(ctx, p.Game, prof.ID)
}
