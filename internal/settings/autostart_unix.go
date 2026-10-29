//go:build unix

package settings

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/portal"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
)

// portalTimeout bounds a portal call, which waits on a dialog the user answers.
const portalTimeout = 5 * time.Minute

// RemoveAutostart deletes the autostart entry that starts Mortar at sign-in.
func RemoveAutostart() error { return applyAutostart(false) }

func applyAutostart(enable bool) error {
	if flatpak.Inside() {
		// The sandbox cannot write the host's autostart folder; the Background portal writes the entry for it.
		ctx, cancel := context.WithTimeout(context.Background(), portalTimeout)
		defer cancel()
		err := portal.SetAutostart(ctx, portal.Autostart{Enable: enable, Command: []string{"mortar"}, Reason: "Start Mortar when you sign in"})
		switch {
		case errors.Is(err, portal.ErrCancelled):
			return errors.New("cancelled in the system dialog")
		case errors.Is(err, portal.ErrNotAllowed):
			return errors.New("starting at sign-in was not allowed")
		}
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg := userdirs.ConfigHome(home)
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		return err
	}
	root, err := os.OpenRoot(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if enable {
		if err := root.Mkdir("autostart", 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		sub, err := root.OpenRoot("autostart")
		if err != nil {
			return err
		}
		defer func() { _ = sub.Close() }()
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		body := autostartEntry(selfexe.Launchable(exe))
		return sub.WriteFile("mortar.desktop", []byte(body), 0o600)
	}
	sub, err := root.OpenRoot("autostart")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = sub.Close() }()
	err = sub.Remove("mortar.desktop")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// autostartEntry is the autostart desktop entry for exe.
func autostartEntry(exe string) string {
	return "[Desktop Entry]\nType=Application\nName=Mortar\nExec=" + desktopentry.Exec(exe) + "\nX-GNOME-Autostart-enabled=true\n"
}
