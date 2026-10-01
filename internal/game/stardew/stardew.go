// Package stardew is the Stardew Valley implementation of game.Game.
package stardew

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/steam"
)

// marker ships in every install on every platform; the launcher name differs per OS.
const marker = "Stardew Valley.dll"

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

var configuredComponents *components.Client

// ConfigureComponents selects the verified component manifest used by this game.
func ConfigureComponents(client *components.Client) { configuredComponents = client }

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
