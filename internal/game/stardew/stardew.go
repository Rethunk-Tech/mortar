// Package stardew is the Stardew Valley implementation of game.Game.
package stardew

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/Rethunk-Tech/mortar/internal/gog"
	"github.com/Rethunk-Tech/mortar/internal/lutris"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launch"
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

// Game is Stardew Valley. The configured component client supplies loader releases; the other fields exist so
// tests can point network and filesystem operations elsewhere.
type Game struct {
	Client       *http.Client
	Components   *components.Client
	ReleasesURL  string
	DownloadBase string
	AssetPattern string
	// CacheDir holds the cached release lookup.
	CacheDir string
	// LogDir holds SMAPI-latest.txt.
	LogDir string
	// Runner, LookPath and LaunchTiming default to the real command, exec.LookPath and 60 s.
	Runner       launch.Runner
	LookPath     func(string) (string, error)
	LaunchTiming launch.Timing
	// DataDir is Mortar's data folder, used to check Flatpak Steam filesystem access; empty means datadir.Dir.
	DataDir string
	// FlatpakShow is `flatpak override --user --show` for Steam; nil means the real command.
	FlatpakShow func() (string, error)
}

// configuredComponents is set by every service that builds a component client, possibly at the same time.
var configuredComponents atomic.Pointer[components.Client]

// ConfigureComponents selects the verified component manifest used by this game.
func ConfigureComponents(client *components.Client) { configuredComponents.Store(client) }

func (Game) ID() string         { return "stardew" }
func (Game) Name() string       { return identity().Name }
func (Game) SteamAppID() string { return identity().SteamAppID }

// GOG names Stardew Valley to the GOG locators.
func (Game) GOG() gog.Game {
	g := identity()
	return gog.Game{ProductID: g.GOG.ProductID, Folder: g.GOG.Folder, Marker: g.Marker}
}

// Lutris names Stardew Valley to the Lutris locator.
func (Game) Lutris() lutris.Game {
	g := identity()
	return lutris.Game{Slug: g.Lutris.Slug, Keyword: g.Lutris.Keyword, Marker: g.Marker}
}
func (Game) LoaderName() string   { return identity().Loader }
func (Game) ModSources() []string { return []string{"nexus", "github"} }

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
