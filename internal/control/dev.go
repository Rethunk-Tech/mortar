package control

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// devAdd puts the mod project folder p.Path in the Develop tab, installing its builds into profile p.Profile (an id or
// a name).
func (s *Services) devAdd(ctx context.Context, p Params) (settings.DevProject, error) {
	id, err := s.devProfile(p)
	if err != nil {
		return settings.DevProject{}, err
	}
	return s.Dev.Add(ctx, p.Game, p.Path, id)
}

// devClone clones the GitHub repository p.ID into the folder p.Path and adds it as devAdd does.
func (s *Services) devClone(ctx context.Context, p Params) (settings.DevProject, error) {
	id, err := s.devProfile(p)
	if err != nil {
		return settings.DevProject{}, err
	}
	return s.Dev.Clone(ctx, p.Game, p.ID, p.Path, id)
}

var errNoDev = errors.New("mod development is unavailable")

// devProfile is the profile p names for a project's builds (an id or a name; empty is none), once development is
// available.
func (s *Services) devProfile(p Params) (string, error) {
	id, err := s.optionalProfile(p.Game, p.Profile)
	// No profile named: the one open in the game, as the Develop tab adds to.
	if p.Profile == "" && s.Settings != nil {
		if open := s.Settings.Get().LastProfile[p.Game]; open != "" {
			if _, getErr := s.Profiles.Get(p.Game, open); getErr == nil {
				id = open
			}
		}
	}
	if err == nil && s.Dev == nil {
		err = errNoDev
	}
	return id, err
}

// devList lists the game's mod projects.
func (s *Services) devList(p Params) ([]settings.DevProject, error) {
	if s.Dev == nil {
		return nil, errNoDev
	}
	return s.Dev.Projects(p.Game), nil
}

// devRemove takes the project whose folder is p.Path, or whose name is p.ID, off the Develop tab; its folder stays.
func (s *Services) devRemove(p Params) (settings.DevProject, error) {
	if s.Dev == nil {
		return settings.DevProject{}, errNoDev
	}
	for _, d := range s.Dev.Projects(p.Game) {
		if d.Dir == p.Path || strings.EqualFold(d.Name, p.ID) {
			return d, s.Dev.Forget(p.Game, d.Dir)
		}
	}
	return settings.DevProject{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("no mod project %s in %s", p.ID, p.Game))
}
