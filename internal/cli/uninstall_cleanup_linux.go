package cli

import (
	"errors"
	"os"

	"github.com/Rethunk-AI/mortar/internal/nxm"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/shortcut"
)

// removePlatformLeftovers removes the sign-in autostart entry, the profile desktop entries and Mortar's own desktop
// entry, icons and file-type registration. The package managers' remove scripts run it for the user who ran them;
// for an AppImage or the portable binary the user runs it.
func removePlatformLeftovers() error {
	errs := []error{settings.RemoveAutostart(), shortcut.RemoveStartMenu()}
	exe, err := os.Executable()
	if err == nil {
		var h *nxm.System
		if h, err = nxm.New(exe); err == nil {
			err = h.RemoveDesktop()
		}
	}
	return errors.Join(append(errs, err)...)
}
