package game

import (
	"errors"
	"fmt"

	"github.com/Rethunk-Tech/go-gameproc"
	"github.com/Rethunk-Tech/mortar/internal/appversion"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
)

// LaunchOptions returns the launch options Steam holds for the game, or "" when none are set or Steam is missing.
func (s *Service) LaunchOptions(id string) (string, error) {
	g, err := Require(id)
	if err != nil {
		return "", err
	}
	st, status := LocateSteam(s.home, s.store.Get())
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
	st, err := s.closedSteam()
	if err != nil {
		return "", err
	}
	exe, err := requireSteamExe(id, dir)
	if err != nil {
		return "", err
	}
	return setLaunchOptions(st, g.SteamAppID(), func(current string) string { return launchWith(exe, current) })
}

// ClearLaunchOption removes the game's loader command from Steam's launch options.
func (s *Service) ClearLaunchOption(id string) (string, error) {
	g, err := Require(id)
	if err != nil {
		return "", err
	}
	st, err := s.closedSteam()
	if err != nil {
		return "", err
	}
	exe, err := requireSteamExe(id, "")
	if err != nil {
		return "", err
	}
	return setLaunchOptions(st, g.SteamAppID(), func(current string) string { return launchWithout(exe, current) })
}

// setLaunchOptions edits st's launch options for appID, telling the player what to do when Steam has none to edit yet.
func setLaunchOptions(st steam.Steam, appID string, merge func(string) string) (string, error) {
	out, err := st.SetLaunchOptions(appID, merge, steam.EditOptions{BackupSuffix: appversion.SteamBackupSuffix})
	if errors.Is(err, steam.ErrNoAppsSection) {
		err = fmt.Errorf("%w; launch the game from Steam once first", err)
	}
	return out, err
}

// closedSteam is the Steam whose config holds launch options, refused while it runs: Steam rewrites its config when it
// exits.
func (s *Service) closedSteam() (steam.Steam, error) {
	st, status := LocateSteam(s.home, s.store.Get())
	if status != steam.Found {
		return steam.Steam{}, errors.New("steam was not found")
	}
	if running, err := gameproc.Processes("/proc", "steam"); err == nil && len(running) > 0 {
		return steam.Steam{}, errors.New("close Steam first: it rewrites its settings when it exits")
	}
	return st, nil
}
