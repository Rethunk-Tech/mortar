package playcheck

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// Service answers the pre-Play check over the services that hold its facts.
type Service struct {
	Problems *problems.Service
	Saves    *savessvc.Service
	Profiles *profile.Service
	Launches *launchsvc.Service
	Settings *settings.Store
}

// Check lists what to tell the player before the profile is played. A failed save scan or history read leaves out
// that part, so the problems still show.
func (s *Service) Check(ctx context.Context, game, profileID string) ([]Group, error) {
	res, err := s.Problems.Problems(ctx, game, profileID)
	if err != nil {
		return nil, err
	}
	upd, err := s.Problems.Updates(ctx, game, profileID)
	if err != nil {
		return nil, err
	}
	prof, err := s.Profiles.Get(game, profileID)
	if err != nil {
		return nil, err
	}
	in := Input{
		ProfileID: profileID, Profile: prof, Problems: res, Updates: upd,
		SmapiNever: s.Settings != nil && s.Settings.Get().SmapiBuilds == settings.SmapiBuildsNever,
		ProfileName: func(id string) string {
			p, err := s.Profiles.Get(game, id)
			if err != nil {
				return ""
			}
			return p.Name
		},
	}
	in.Installed, _ = s.Profiles.Mods(game, profileID)
	if s.Saves != nil {
		if fit, _, err := s.Saves.LastSaveGap(ctx, game, profileID); err == nil {
			in.Fit = fit
		}
	}
	if s.Launches != nil {
		if since, err := s.Launches.LastRunStarted(game, profileID); err == nil {
			if diff, err := s.Profiles.ChangesSince(game, profileID, since); err == nil {
				in.Changes = &diff
			}
		}
	}
	return Groups(in), nil
}
