package nxm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Launchable is the path a desktop entry, shortcut or handler should run to start this Mortar: $APPIMAGE when exe
// runs from that AppImage's mount, which vanishes on exit, else exe. A child started from an AppImage inherits
// APPIMAGE, hence the check that exe lives under $APPDIR.
func Launchable(exe string) string {
	img, mount := os.Getenv("APPIMAGE"), os.Getenv("APPDIR")
	if img == "" || mount == "" || !strings.HasPrefix(exe, filepath.Clean(mount)+string(filepath.Separator)) {
		return exe
	}
	if info, err := fsx.Stat(img); err != nil || !info.Mode().IsRegular() {
		return exe
	}
	return img
}
