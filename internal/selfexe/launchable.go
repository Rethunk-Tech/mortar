// Package selfexe resolves the path other processes should run to start this Mortar.
package selfexe

import (
	"os"

	"github.com/Rethunk-Tech/go-safefs"
)

// Launchable is the path a desktop entry, shortcut or handler should run to start this Mortar: $APPIMAGE when exe
// runs from that AppImage's mount, which vanishes on exit, else exe. A child started from an AppImage inherits
// APPIMAGE, hence the check that exe lives under $APPDIR.
func Launchable(exe string) string {
	img, mount := os.Getenv("APPIMAGE"), os.Getenv("APPDIR")
	if img == "" || mount == "" || !safefs.UnderRoot(mount, exe) {
		return exe
	}
	if !safefs.IsFile(img) {
		return exe
	}
	return img
}
