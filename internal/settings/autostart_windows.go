//go:build windows

package settings

import (
	"context"
	"os"

	"github.com/Rethunk-Tech/go-desktopenv/autostart"
)

// systemAutostart writes or removes the HKCU Run value; tests replace it.
var systemAutostart = func(enable bool) error {
	if !enable {
		return autostart.Disable(context.Background(), "Mortar")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return autostart.Enable(context.Background(), autostart.Entry{ID: "Mortar", Argv: []string{exe}})
}

func applyAutostart(enable bool) error { return systemAutostart(enable) }

// RemoveAutostart deletes the Run value that starts Mortar at sign-in.
func RemoveAutostart() error { return applyAutostart(false) }
