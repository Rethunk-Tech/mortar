// Package game is the registry of supported games: the Game interface, its Stardew implementation and display-only entries.
package game

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/game/stardew"
	"github.com/Rethunk-AI/mortar/internal/steam"
)

const artPrefix = "/steam-art/"

// Game is everything that differs per game.
type Game interface {
	ID() string
	Name() string
	SteamAppID() string
	LoaderName() string
	ModSources() []string
	// ValidInstall reports why dir is not this game's install folder, or nil.
	ValidInstall(dir string) error
	// Discover prefers a still-valid override folder over Steam (st is nil without Steam) and returns "" when not installed.
	Discover(override string, st *steam.Steam) (string, error)
}

var games = []Game{stardew.Game{}}

// comingLater lists games Game Select shows before they have an implementation.
type listing struct{ id, name, appID, loader string }

var comingLater = []listing{
	{"lethal", "Lethal Company", "1966720", "BepInEx 5"},
}

func byID(id string) Game {
	for _, g := range games {
		if g.ID() == id {
			return g
		}
	}
	return nil
}

// Valid reports whether id names a listed game.
func Valid(id string) bool {
	return byID(id) != nil || slices.ContainsFunc(comingLater, func(c listing) bool { return c.id == id })
}

func knownApp(appID string) bool {
	if _, err := strconv.ParseUint(appID, 10, 32); err != nil {
		return false
	}
	return slices.ContainsFunc(games, func(g Game) bool { return g.SteamAppID() == appID }) ||
		slices.ContainsFunc(comingLater, func(c listing) bool { return c.appID == appID })
}

// ArtMiddleware serves GET /steam-art/<appid> for listed games only, from Steam's cached hero image.
func ArtMiddleware(home string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := strings.CutPrefix(r.URL.Path, artPrefix)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if r.Method != http.MethodGet || !knownApp(id) {
				http.NotFound(w, r)
				return
			}
			st, status := steam.Locate(home)
			if status != steam.Found {
				http.NotFound(w, r)
				return
			}
			path := st.HeroArt(id)
			if path == "" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			http.ServeFile(w, r, path)
		})
	}
}
