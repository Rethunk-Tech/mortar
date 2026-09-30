// Package stardew is the Stardew Valley implementation of game.Game.
package stardew

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/steam"
)

// marker ships in every install on every platform; the launcher name differs per OS.
const marker = "Stardew Valley.dll"

// Game is Stardew Valley. The zero value talks to GitHub and uses Mortar's data and the user's config folder;
// the fields exist so tests can point it elsewhere.
type Game struct {
	Client       *http.Client
	ReleasesURL  string
	DownloadBase string
	// CacheDir holds the cached release lookup.
	CacheDir string
	// LogDir holds SMAPI-latest.txt.
	LogDir string
}

func (Game) ID() string           { return "stardew" }
func (Game) Name() string         { return "Stardew Valley" }
func (Game) SteamAppID() string   { return "413150" }
func (Game) LoaderName() string   { return "SMAPI" }
func (Game) ModSources() []string { return []string{"nexus", "github"} }

// ValidInstall reports why dir is not a Stardew Valley install folder.
func (Game) ValidInstall(dir string) error {
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
