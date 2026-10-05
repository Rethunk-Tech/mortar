package profile

import (
	"fmt"

	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func (s *Store) launchSettingsLoader(game, id string) (loader.WithLaunchSettings, string, error) {
	dir, err := s.ProfileDir(game, id)
	if err != nil {
		return nil, "", err
	}
	l, ok := gamereg.LoaderOf(game, s.LoaderID(game, id))
	if !ok {
		return nil, dir, nil
	}
	w, _ := l.(loader.WithLaunchSettings)
	return w, dir, nil
}

// LoaderLaunchSettings are the launch options the profile's loader keeps in the profile; none when it has no such options.
func (s *Service) LoaderLaunchSettings(game, id string) ([]loader.LaunchSetting, error) {
	w, dir, err := s.store.launchSettingsLoader(game, id)
	if err != nil || w == nil {
		return nil, err
	}
	return w.LaunchSettings(dir)
}

// SetLoaderLaunchSetting changes one of the profile's loader launch options.
func (s *Service) SetLoaderLaunchSetting(game, id, setting, value string) error {
	w, dir, err := s.store.launchSettingsLoader(game, id)
	if err != nil {
		return err
	}
	if w == nil {
		return fmt.Errorf("the profile's loader has no launch settings")
	}
	return w.SetLaunchSetting(dir, setting, value)
}
