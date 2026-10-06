package game

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Catalog path roles.
const (
	PathSaves              = "saves"
	PathStartupPreferences = "startupPreferences"
	// PathUnityLog is the Unity player log of a game built on Unity.
	PathUnityLog = "unityLog"
)

// HasPath reports whether the catalog gives game id a path for role: a game has a save folder or startup
// preferences when its entry says where they live.
func HasPath(id, role string) bool {
	g, ok := catalogGame(id)
	_, has := g.Paths[role]
	return ok && has
}

// HasSaves reports whether game id has a save folder.
func HasSaves(id string) bool { return HasPath(id, PathSaves) }

// HasStartupSettings reports whether game id reads startup preferences from a file outside the install.
func HasStartupSettings(id string) bool { return HasPath(id, PathStartupPreferences) }

// PathFor resolves game id's path for role in the install pin names (the selected install when empty).
func PathFor(home string, s settings.Settings, id, pin, role string) (string, error) {
	g, ok := catalogGame(id)
	t, has := g.Paths[role]
	if !ok || !has {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("game %q has no %s path", id, role))
	}
	in, err := ResolveInstall(home, s, id, pin)
	if err != nil {
		return "", err
	}
	return runtime.Resolve(in.runtime(g, home), t)
}

// SavesDir is game id's saves folder for the install pin names (the selected install when empty).
func SavesDir(home string, s settings.Settings, id, pin string) (string, error) {
	return PathFor(home, s, id, pin, PathSaves)
}

// StartupPreferencesPath is the file holding game id's startup preferences.
func StartupPreferencesPath(home string, s settings.Settings, id, pin string) (string, error) {
	return PathFor(home, s, id, pin, PathStartupPreferences)
}

// SaveFiles are the patterns naming game id's save files in its saves folder; none means each save is a folder.
func SaveFiles(id string) []string {
	g, _ := catalogGame(id)
	return g.SaveFiles
}
