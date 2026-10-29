package doctor

import (
	"os"
	"slices"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// userHome is replaced by tests.
var userHome = os.UserHomeDir

// flatpakLibraryChecks reports Steam libraries the Flatpak cannot see. The manifest grants the usual places; a
// library anywhere else reads as missing from inside the sandbox, so its games look not installed.
func flatpakLibraryChecks(steamRoots []string) []Check {
	if !flatpak.Inside() {
		return nil
	}
	home, err := userHome()
	if err != nil {
		return nil
	}
	var seen []string
	var checks []Check
	for _, s := range steam.LocateAll(home, steamRoots...) {
		libs, err := s.Libraries()
		if err != nil {
			continue
		}
		for _, lib := range libs {
			if slices.Contains(seen, lib) || safefs.IsDir(lib) {
				continue
			}
			seen = append(seen, lib)
			checks = append(checks, Check{
				ID:     "flatpakLibrary:" + lib,
				Status: Warn,
				Detail: "Steam library " + lib + " is not reachable: unmounted, or outside the folders Mortar's Flatpak may read",
				Code:   "flatpakLibrary",
				Params: params("path", lib),
				Fix:    "flatpak override --user --filesystem=" + lib + " " + appversion.AppID + ", then restart Mortar",
			})
		}
	}
	return checks
}
