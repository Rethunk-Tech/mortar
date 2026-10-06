package launchsvc

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// menuMarker in the profile's startup folder asks the next launch to go straight to the main menu, whatever the
// skipIntro setting says; the launch consumes it. A crash-check step sets it, since nobody answers the game's
// start-up choices while it runs.
const menuMarker = ".menu-next-launch"

// armIntroSkip sets the intro request of the profile's loader for this launch: to the menu when a crash check asked,
// else the boot animations when the skipIntro setting is on, else none.
func (s *Service) armIntroSkip(g game.Game, l loader.Loader, profileID, dir string) error {
	skipper, ok := l.(loader.IntroSkipper)
	if !ok {
		return nil
	}
	err := os.Remove(filepath.Join(dir, startupDir, menuMarker))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	skip := loader.IntroPlay
	switch {
	case err == nil:
		skip = loader.IntroToMenu
	case s.settings != nil && settings.ResolveAt(s.settings.Get(), "skipIntro", settings.Scope{Game: g.ID(), Install: installOf(g), Profile: profileID}, launchOverrides(s.profiles, g.ID(), profileID)) == "true":
		skip = loader.IntroAnimations
	}
	return skipper.ArmIntroSkip(dir, skip)
}

// launchToMenu asks the profile's next launch to go straight to the main menu. The returned func withdraws an
// unconsumed request, so a crash-check launch that failed before starting leaves the player's next Play alone.
func (s *Service) launchToMenu(gameID, profileID string) (func(), error) {
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, startupDir, menuMarker)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := datadir.WriteFile(path, nil, 0o600); err != nil {
		return nil, err
	}
	return func() { _ = os.Remove(path) }, nil
}
