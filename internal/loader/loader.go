// Package loader holds the types shared by every game's mod loader implementation.
package loader

import (
	"strconv"
	"strings"
)

// Step is one stage of a loader install, reported as it completes.
type Step string

const (
	StepDownloaded Step = "downloaded"
	StepFiles      Step = "files"
	StepLauncher   Step = "launcher"
	StepBundled    Step = "bundled"
)

// Status is a game's loader state on disk plus the newest release, when it could be fetched.
type Status struct {
	Installed bool `json:"installed"`
	// Broken means loader files exist but the game's launcher no longer starts the loader, as after a game update.
	Broken      bool   `json:"broken"`
	Version     string `json:"version"`
	GameVersion string `json:"gameVersion"`
	// Latest is empty when the release lookup failed.
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
}

// Bundled receives the loader's own mods, extracted into modsDir, while the installer's files still exist.
type Bundled func(version, modsDir string) error

// Newer reports whether version a is newer than b, comparing dotted numbers.
func Newer(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range max(len(as), len(bs)) {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}
