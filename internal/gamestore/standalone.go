package gamestore

import (
	"path/filepath"
	goruntime "runtime"

	"github.com/Rethunk-Tech/go-desktopenv/userdirs"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/gog"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
)

const (
	standaloneKey = "standalone"
	flatpakKey    = "flatpak"
)

// gameDir is dir (or its "game" folder) when it holds the game's marker or its native Linux build's executable.
func gameDir(dir string, g components.GameInfo) string {
	if found := gog.GameDir(dir, g.MarkerPath()); found != "" {
		return found
	}
	if g.LinuxMarker != "" && goruntime.GOOS == "linux" {
		return gog.GameDir(dir, g.LinuxMarker)
	}
	return ""
}

// standaloneStore finds a game sold outside any store app in the folders its own installer uses.
type standaloneStore struct{}

func (standaloneStore) Key() string { return standaloneKey }

func (standaloneStore) Launchers(string) []LauncherSpec { return nil }

func (standaloneStore) Discover(home string, _ map[string][]string, g components.GameInfo) []Install {
	var out []Install
	for _, t := range g.Stores.Standalone.Folders {
		dir, err := runtime.Resolve(runtime.Install{Home: home, Platform: goruntime.GOOS}, t)
		if err != nil {
			continue
		}
		if found := gameDir(dir, g); found != "" {
			out = append(out, Install{Store: StoreStandalone, Dir: found})
		}
	}
	return out
}

// flatpakRoots are the per-user and system-wide Flatpak installations.
func flatpakRoots(home string) []string {
	return []string{filepath.Join(userdirs.HostDataHome(home), "flatpak"), "/var/lib/flatpak"}
}

// flatpakStore finds a game installed as a Flatpak app, in the app's current deployment.
type flatpakStore struct{}

func (flatpakStore) Key() string { return flatpakKey }

func (flatpakStore) Launchers(string) []LauncherSpec { return nil }

func (flatpakStore) Discover(home string, _ map[string][]string, g components.GameInfo) []Install {
	if goruntime.GOOS != "linux" {
		return nil
	}
	var out []Install
	for _, root := range flatpakRoots(home) {
		dir := filepath.Join(root, "app", g.Stores.Flatpak.App, "current", "active", "files", filepath.FromSlash(g.Stores.Flatpak.Folder))
		if found := gameDir(dir, g); found != "" {
			out = append(out, Install{Store: StoreFlatpak, Dir: found})
		}
	}
	return out
}
