package profile

import (
	"fmt"

	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
)

// LoaderID is the id of the loader the profile runs: its own choice, else the game's primary loader, "" when the game
// has none.
func (s *Store) LoaderID(gameID, id string) string {
	if p, err := s.read(gameID, id); err == nil && p.Loader != "" {
		return p.Loader
	}
	if l, ok := gamereg.PrimaryLoader(gameID); ok {
		return l.ID()
	}
	return ""
}

// SetLoader makes the profile run the game's loader with this id; "" returns it to the primary loader. It never
// touches mods/, so a running game does not block it.
func (s *Store) SetLoader(gameID, id, loaderID string) (Profile, error) {
	if loaderID != "" {
		if _, ok := gamereg.LoaderOf(gameID, loaderID); !ok {
			return Profile{}, fmt.Errorf("game %q has no loader %q", gameID, loaderID)
		}
	}
	return s.update(gameID, id, func(p *Profile, _ string) error {
		p.Loader = loaderID
		return nil
	})
}
