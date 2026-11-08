//go:build unix

package settings

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/autostart"
	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
	"github.com/Rethunk-Tech/go-desktopenv/portal"
)

// autostartID is the autostart file's stem; earlier releases wrote mortar.desktop, so it stays.
const autostartID = "mortar"

// systemAutostart writes or removes the entry, through the Background portal inside a Flatpak, bounded because the
// portal waits on a dialog the user answers; tests replace it.
var systemAutostart = func(enable bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if !enable {
		return autostart.Disable(ctx, autostartID)
	}
	argv := []string{"mortar"}
	if !flatpak.Inside() {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		argv = []string{hostenv.Launchable(exe)}
	}
	return autostart.Enable(ctx, autostart.Entry{ID: autostartID, Name: "Mortar", Reason: "Start Mortar when you sign in", Argv: argv})
}

// RemoveAutostart deletes the autostart entry that starts Mortar at sign-in.
func RemoveAutostart() error { return applyAutostart(false) }

func applyAutostart(enable bool) error {
	err := systemAutostart(enable)
	switch {
	case errors.Is(err, portal.ErrCancelled):
		return errors.New("cancelled in the system dialog")
	case errors.Is(err, portal.ErrNotAllowed):
		return errors.New("starting at sign-in was not allowed")
	}
	return err
}
