package profile

import gamereg "github.com/Rethunk-Tech/mortar/internal/game"

// LaunchOptions returns the profile's extra SMAPI arguments as stored.
func (s *Store) LaunchOptions(game, id string) (string, error) {
	p, err := s.read(game, id)
	if err != nil {
		return "", err
	}
	return p.LaunchOptions, nil
}

// SetLaunchOptions replaces a profile's extra SMAPI arguments. It never touches mods/, so a running
// game does not block it.
func (s *Store) SetLaunchOptions(game, id, options string) (Profile, error) {
	if _, err := gamereg.ParseLaunchOptions(game, s.LoaderID(game, id), options); err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.LaunchOptions = options
		return nil
	})
}
