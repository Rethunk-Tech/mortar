// Package lethal is the Lethal Company implementation of game.Game: its identity and install folder. BepInEx and the
// deploy method do the rest.
package lethal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

const id = "lethal-company"

// identity is the game's catalog entry; the bundled manifest always carries it.
func identity() components.GameInfo {
	g, _ := components.BundledGame(id)
	return g
}

// Game is Lethal Company.
type Game struct{}

func (Game) ID() string         { return id }
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

// GameProcesses are the game's executables.
func (Game) GameProcesses() []string { return []string{"Lethal Company.exe", "Lethal Company"} }

// ValidInstall reports why dir is not a Lethal Company install folder.
func (Game) ValidInstall(dir string) error {
	marker := identity().Marker
	st, err := os.Stat(filepath.Join(dir, marker))
	if err != nil || !st.Mode().IsRegular() {
		return fmt.Errorf("%q is not a Lethal Company folder: it has no %s", dir, marker)
	}
	return nil
}

// Discover returns override when it is still a valid install, else the Steam install; "" when neither exists.
func (g Game) Discover(override string, st *steam.Steam) (string, error) {
	if override != "" && g.ValidInstall(override) == nil {
		return override, nil
	}
	if st == nil {
		return "", nil
	}
	return st.InstallDir(g.SteamAppID())
}
