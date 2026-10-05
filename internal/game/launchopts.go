package game

import (
	"errors"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/launch"

	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// LaunchOptions returns the launch options Steam holds for the game, or "" when none are set or Steam is missing.
func (s *Service) LaunchOptions(id string) (string, error) {
	g, err := Require(id)
	if err != nil {
		return "", err
	}
	st, status := steam.Locate(s.home)
	if status != steam.Found {
		return "", nil
	}
	return st.LaunchOptions(g.SteamAppID())
}

// SetLaunchOption writes the launch options that make Steam start the game's loader, keeping the user's own options,
// and returns them. Steam rewrites its config when it exits, so this refuses while Steam is running.
func (s *Service) SetLaunchOption(id string) (string, error) {
	g, err := Require(id)
	if err != nil {
		return "", err
	}
	dir, err := InstallDir(s.home, s.store.Get(), id)
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("%s is not installed", g.Name())
	}
	st, status := steam.Locate(s.home, roots(s.store.Get(), LauncherSteam)...)
	if status != steam.Found {
		return "", errors.New("steam was not found")
	}
	if running, err := launch.Processes("/proc", "steam"); err == nil && len(running) > 0 {
		return "", errors.New("close Steam first: it rewrites its settings when it exits")
	}
	exe, err := requireSteamExe(id, dir)
	if err != nil {
		return "", err
	}
	return st.SetLaunchOptions(g.SteamAppID(), func(current string) string { return launchWith(exe, current) })
}

// ClearLaunchOption removes the game's loader command from Steam's launch options.
func (s *Service) ClearLaunchOption(id string) (string, error) {
	g, err := Require(id)
	if err != nil {
		return "", err
	}
	st, status := steam.Locate(s.home, roots(s.store.Get(), LauncherSteam)...)
	if status != steam.Found {
		return "", errors.New("steam was not found")
	}
	if running, err := launch.Processes("/proc", "steam"); err == nil && len(running) > 0 {
		return "", errors.New("close Steam first: it rewrites its settings when it exits")
	}
	exe, err := requireSteamExe(id, "")
	if err != nil {
		return "", err
	}
	return st.SetLaunchOptions(g.SteamAppID(), func(current string) string { return launchWithout(exe, current) })
}
