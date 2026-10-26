package control

import (
	"context"
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// devAdd puts the mod project folder p.Path in the Develop tab, installing its builds into profile p.Profile.
func (s *Services) devAdd(ctx context.Context, p Params) (settings.DevProject, error) {
	if s.Dev == nil {
		return settings.DevProject{}, errors.New("mod development is unavailable")
	}
	return s.Dev.Add(ctx, p.Game, p.Path, p.Profile)
}

// devClone clones the GitHub repository p.ID into the folder p.Path and adds it as devAdd does.
func (s *Services) devClone(ctx context.Context, p Params) (settings.DevProject, error) {
	if s.Dev == nil {
		return settings.DevProject{}, errors.New("mod development is unavailable")
	}
	return s.Dev.Clone(ctx, p.Game, p.ID, p.Path, p.Profile)
}
