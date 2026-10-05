package profile

import (
	"errors"

	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

var errNoLaunchSettings = errors.New("the profile's loader has no launch settings")

// withLaunchSettings runs f with the profile's loader and folder, or fails when the loader keeps no launch options.
func (s *Store) withLaunchSettings(game, id string, f func(w loader.WithLaunchSettings, dir string) error) error {
	dir, err := s.ProfileDir(game, id)
	if err != nil {
		return err
	}
	l, ok := gamereg.LoaderOf(game, s.LoaderID(game, id))
	if !ok {
		return errNoLaunchSettings
	}
	w, ok := l.(loader.WithLaunchSettings)
	if !ok {
		return errNoLaunchSettings
	}
	return f(w, dir)
}

// LoaderLaunchSettings are the launch options the profile's loader keeps in the profile; none when it has no such options.
func (s *Service) LoaderLaunchSettings(game, id string) (out []loader.LaunchSetting, err error) {
	err = s.store.withLaunchSettings(game, id, func(w loader.WithLaunchSettings, dir string) (e error) {
		out, e = w.LaunchSettings(dir)
		return e
	})
	if errors.Is(err, errNoLaunchSettings) {
		return nil, nil
	}
	return out, err
}

// SetLoaderLaunchSetting changes one of the profile's loader launch options.
func (s *Service) SetLoaderLaunchSetting(game, id, setting, value string) error {
	return s.store.withLaunchSettings(game, id, func(w loader.WithLaunchSettings, dir string) error {
		return w.SetLaunchSetting(dir, setting, value)
	})
}
