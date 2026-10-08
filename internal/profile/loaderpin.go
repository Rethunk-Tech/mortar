package profile

import (
	"fmt"

	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// LoaderID is the id of the loader the profile runs: its own choice, else the game's primary loader, "" when the game
// has none.
func (s *Store) LoaderID(gameID, id string) string {
	p, _ := s.read(gameID, id)
	return p.LoaderFor(gameID)
}

// LoaderFor is LoaderID for a profile already in hand.
func (p Profile) LoaderFor(gameID string) string {
	if p.Loader != "" {
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

// InstalledLoader is the id of the loader the profile runs when that loader is installed where it lives (in the
// profile, or in the game folder for a loader the game folder holds), "" otherwise. It reads only the disk.
func (s *Store) InstalledLoader(gameID, id string) string {
	l, ok := gamereg.LoaderOf(gameID, s.LoaderID(gameID, id))
	if !ok {
		return ""
	}
	t := loader.Target{Game: gameID}
	if _, perProfile := l.(loader.InProfile); perProfile {
		dir, err := s.ProfileDir(gameID, id)
		if err != nil {
			return ""
		}
		t.ProfileDir = dir
	} else {
		if s.settings == nil {
			return ""
		}
		dir, err := gamereg.InstallDir(s.home, s.settings.Get(), gameID)
		if err != nil || dir == "" {
			return ""
		}
		t.InstallDir = dir
	}
	st, err := l.Status(t)
	if err != nil || !st.Installed || st.Broken {
		return ""
	}
	return l.ID()
}

// InstalledLoader is the id of the profile's loader when it is installed, "" otherwise.
func (s *Service) InstalledLoader(game, id string) string { return s.store.InstalledLoader(game, id) }
