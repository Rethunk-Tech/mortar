package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// uninstallCleanup undoes what Mortar wrote outside its install folder: the start-at-sign-in entry, the profile
// shortcuts, the profiles added to Steam and, on Linux, its desktop entry, icons and file types. The Windows
// uninstaller and the Linux packages' remove scripts run it before the program goes, which is why it is not listed in
// the usage text. Profiles and mods are left alone.
func (c *cmd) uninstallCleanup() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe = selfexe.Launchable(exe)
	errs := []error{removePlatformLeftovers()}
	if home, err := os.UserHomeDir(); err == nil {
		errs = append(errs, c.removeSteamShortcuts(home, exe, settings.Peek().LauncherRoots[game.LauncherSteam]))
	} else {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (c *cmd) removeSteamShortcuts(home, exe string, steamRoots []string) error {
	var errs []error
	removed := 0
	for _, st := range steam.LocateAll(home, steamRoots...) {
		n, err := st.RemoveShortcuts(exe, "--play=")
		removed += n
		errs = append(errs, err)
	}
	if removed > 0 {
		fmt.Fprintf(c.out, "Removed %d Mortar shortcuts from Steam\n", removed)
	}
	return errors.Join(errs...)
}
