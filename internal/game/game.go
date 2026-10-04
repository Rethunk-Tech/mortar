// Package game is the registry of supported games: the Game interface, its Stardew implementation and display-only entries.
package game

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/gog"
	"github.com/Rethunk-AI/mortar/internal/lutris"

	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/game/stardew"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/loader"
	"github.com/Rethunk-AI/mortar/internal/steam"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

const artPrefix = "/steam-art/"

// ArtURL is where the frontend loads Steam's hero art for appID from; ArtMiddleware serves it.
func ArtURL(appID string) string { return artPrefix + appID }

// Game is everything that differs per game.
type Game interface {
	Identity
	Installs
	Loaders
	Launcher
}

// Identity names a game and where its process and mods come from.
type Identity interface {
	ID() string
	Name() string
	SteamAppID() string
	LoaderName() string
	ModSources() []string
	// ProcessName is the loader's executable, the process a running profile is found by.
	ProcessName() string
	// GameProcesses are the executables of the game with or without its loader; any of them running blocks a
	// loader install, whoever started it.
	GameProcesses() []string
	// GOG and Lutris describe the game to those launchers' locators; a game a store does not sell returns the zero value.
	GOG() gog.Game
	Lutris() lutris.Game
}

// Installs finds and validates a game's install folder.
type Installs interface {
	// ValidInstall reports why dir is not this game's install folder, or nil.
	ValidInstall(dir string) error
	// Discover prefers a still-valid override folder over Steam (st is nil without Steam) and returns "" when not installed.
	Discover(override string, st *steam.Steam) (string, error)
}

// Loaders manages the game's mod loader.
type Loaders interface {
	// LoaderStatus reports the loader's state in the install dir; recorded is the version Mortar installed, or "".
	LoaderStatus(dir, recorded string) loader.Status
	// LatestLoader returns the newest stable loader version.
	LatestLoader(ctx context.Context) (string, error)
	// CopyBundled copies the loader's own mods that are already in the install dir into dst, for a loader
	// installed outside Mortar.
	CopyBundled(dir, dst string) error
	// InstallLoader installs or updates the loader in dir and returns its version.
	InstallLoader(ctx context.Context, dir string, bundled loader.Bundled, progress func(loader.Step)) (string, error)
}

// Launcher starts a game.
type Launcher interface {
	// Launch starts the profile's mods folder and returns once the game has started, sending the loader's log
	// lines to onLines in batches until ctx is done. It returns launch.ErrNoSteam when there is no Steam and
	// req.Direct is false. A vanilla request waits for a game process instead of a loader log.
	Launch(ctx context.Context, req launch.Request, onLines func([]string)) error
	// LogFile is the loader's log, which outlives the game.
	LogFile() (string, error)
	// SteamLaunchForcesLoader reports whether Steam's launch options start the loader instead of the game.
	SteamLaunchForcesLoader(options string) bool
	// SteamLaunchWithLoader returns current launch options changed so Steam starts the loader in dir, keeping the
	// user's own options.
	SteamLaunchWithLoader(dir, current string) string
	// SteamLaunchWithoutLoader returns current launch options with the loader command removed.
	SteamLaunchWithoutLoader(current string) string
}

var games = []Game{&stardew.Game{}}

// ConfigureComponents gives implemented games the verified component manifest used for loader installs.
func ConfigureComponents(client *components.Client) {
	stardew.ConfigureComponents(client)
}

// comingLater lists games Game Select shows before they have an implementation.
type listing struct{ id, name, appID, loader string }

var comingLater = []listing{
	{"lethal", "Lethal Company", "1966720", "BepInEx 5"},
}

// Find returns the implemented game with this id, or nil.
func Find(id string) Game {
	for _, g := range games {
		if g.ID() == id {
			return g
		}
	}
	return nil
}

// Require returns the implemented game with this id, or a NotFound error.
func Require(id string) (Game, error) {
	g := Find(id)
	if g == nil {
		return nil, usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", id))
	}
	return g, nil
}

// Valid reports whether id names a listed game.
func Valid(id string) bool {
	return Find(id) != nil || slices.ContainsFunc(comingLater, func(c listing) bool { return c.id == id })
}

// listedApp returns the listing's own copy of appID, so values built from it never carry request text.
func listedApp(appID string) (string, bool) {
	if _, err := strconv.ParseUint(appID, 10, 32); err != nil {
		return "", false
	}
	for _, g := range games {
		if g.SteamAppID() == appID {
			return g.SteamAppID(), true
		}
	}
	for _, c := range comingLater {
		if c.appID == appID {
			return c.appID, true
		}
	}
	return "", false
}

// heroCDN is Steam's public copy of a game's hero image, the same file Steam caches locally.
const heroCDN = "https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_hero.jpg"

// ArtMiddleware serves GET /steam-art/<appid> for listed games only, from Steam's cached hero image, and redirects
// to Steam's public copy when no local Steam has cached it, so a cover list never shows a broken image.
func ArtMiddleware(home string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := strings.CutPrefix(r.URL.Path, artPrefix)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			listed, ok := listedApp(id)
			if r.Method != http.MethodGet || !ok {
				http.NotFound(w, r)
				return
			}
			path := ""
			for _, one := range steam.LocateAll(home) {
				path = one.HeroArt(id)
				if path != "" {
					break
				}
			}
			if path == "" {
				http.Redirect(w, r, fmt.Sprintf(heroCDN, listed), http.StatusFound)
				return
			}
			art, err := fsx.Open(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer func() { _ = art.Close() }()
			info, err := art.Stat()
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			http.ServeContent(w, r, "", info.ModTime(), art)
		})
	}
}
