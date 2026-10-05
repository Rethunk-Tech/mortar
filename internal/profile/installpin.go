package profile

import (
	"github.com/Rethunk-Tech/mortar/internal/game"
)

// Install returns the id of the install the profile is pinned to, or "" for the game's selected install.
func (s *Store) Install(gameID, id string) (string, error) {
	p, err := s.read(gameID, id)
	if err != nil {
		return "", err
	}
	return p.Install, nil
}

// SetInstall pins the profile to the game install with the given id; "" unpins it. It never touches mods/, so a
// running game does not block it.
func (s *Store) SetInstall(gameID, id, install string) (Profile, error) {
	if install != "" && s.settings != nil {
		if _, err := game.ResolveInstall(s.home, s.settings.Get(), gameID, install); err != nil {
			return Profile{}, err
		}
	}
	return s.update(gameID, id, func(p *Profile, _ string) error {
		p.Install = install
		return nil
	})
}

// InstallOf is the pin of a profile that may not exist: empty when it cannot be read, so callers fall back to the
// selected install.
func (s *Store) InstallOf(gameID, id string) string {
	install, _ := s.Install(gameID, id)
	return install
}
