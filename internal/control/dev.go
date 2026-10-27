package control

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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

// devList lists the game's mod projects.
func (s *Services) devList(p Params) ([]settings.DevProject, error) {
	if s.Dev == nil {
		return nil, errors.New("mod development is unavailable")
	}
	return s.Dev.Projects(p.Game), nil
}

// devRemove takes the project whose folder is p.Path, or whose name is p.ID, off the Develop tab; its folder stays.
func (s *Services) devRemove(p Params) (settings.DevProject, error) {
	if s.Dev == nil {
		return settings.DevProject{}, errors.New("mod development is unavailable")
	}
	for _, d := range s.Dev.Projects(p.Game) {
		if d.Dir == p.Path || strings.EqualFold(d.Name, p.ID) {
			return d, s.Dev.Forget(p.Game, d.Dir)
		}
	}
	return settings.DevProject{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("no mod project %s in %s", p.ID, p.Game))
}
