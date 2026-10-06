package doctor

import (
	"os"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// userHome is replaced by tests.
var userHome = os.UserHomeDir

// flatpakLibraryChecks reports Steam libraries the Flatpak cannot see. The manifest grants the usual places; a
// library anywhere else reads as missing from inside the sandbox, so its games look not installed.
func flatpakLibraryChecks() []Check {
	if !sandbox.InFlatpak() {
		return nil
	}
	home, err := userHome()
	if err != nil {
		return nil
	}
	var seen []string
	var checks []Check
	for _, s := range steam.LocateAll(home) {
		libs, err := s.Libraries()
		if err != nil {
			continue
		}
		for _, lib := range libs {
			if slices.Contains(seen, lib) || fsx.IsDir(lib) {
				continue
			}
			seen = append(seen, lib)
			checks = append(checks, Check{
				ID:     "flatpakLibrary:" + lib,
				Status: Warn,
				Detail: "Steam library " + lib + " is not reachable: unmounted, or outside the folders Mortar's Flatpak may read",
				Fix:    "flatpak override --user --filesystem=" + lib + " " + sandbox.AppID + ", then restart Mortar",
			})
		}
	}
	return checks
}
