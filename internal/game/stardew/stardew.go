// Package stardew is the Stardew Valley implementation of game.Game: its identity, install folder and launch options.
package stardew

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// identity is Stardew Valley's names and store ids from the verified component manifest, which always carries them
// because the bundled manifest does.
func identity() components.GameInfo {
	if c := configuredComponents.Load(); c != nil {
		if g, ok := c.Game("stardew"); ok {
			return g
		}
	}
	g, _ := components.BundledGame("stardew")
	return g
}

// Game is Stardew Valley.
type Game struct{}

// configuredComponents is set by every service that builds a component client, possibly at the same time.
var configuredComponents atomic.Pointer[components.Client]

// ConfigureComponents selects the verified component manifest used by this game.
func ConfigureComponents(client *components.Client) { configuredComponents.Store(client) }

func (Game) ID() string         { return "stardew" }
func (Game) Name() string       { return identity().Name }
func (Game) SteamAppID() string { return identity().SteamAppID() }

func (Game) ModSources() []string {
	sources := identity().Sources
	ids := make([]string, len(sources))
	for i, s := range sources {
		ids[i] = s.ID
	}
	return ids
}

// GameProcesses are the game's native executables; the launcher names are what SMAPI's install leaves behind.
func (Game) GameProcesses() []string {
	return []string{"Stardew Valley", "StardewValley", "StardewValley-original", "StardewValley.bin.x86_64"}
}

// ValidInstall reports why dir is not a Stardew Valley install folder.
func (Game) ValidInstall(dir string) error {
	marker := identity().Marker
	st, err := os.Stat(filepath.Join(dir, marker))
	if err != nil || !st.Mode().IsRegular() {
		return fmt.Errorf("%q is not a Stardew Valley folder: it has no %s", dir, marker)
	}
	return nil
}

// Discover returns override when it is still a valid install, else the Steam install; "" when neither exists.
// st is nil when no Steam was found.
func (g Game) Discover(override string, st *steam.Steam) (string, error) {
	if override != "" && g.ValidInstall(override) == nil {
		return override, nil
	}
	if st == nil {
		return "", nil
	}
	return st.InstallDir(g.SteamAppID())
}
