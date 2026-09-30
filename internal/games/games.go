// Package games is the registry of games Mortar supports and their Steam state.
package games

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/steam"
)

const artPrefix = "/steam-art/"

type def struct {
	id, name, appID, loader string
	available               bool
}

var registry = []def{
	{"stardew", "Stardew Valley", "413150", "SMAPI", true},
	{"lethal", "Lethal Company", "1966720", "BepInEx 5", false},
}

// GameInfo is one registry row with its Steam state.
type GameInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AppID      string `json:"appId"`
	Loader     string `json:"loader"`
	Available  bool   `json:"available"`
	Installed  bool   `json:"installed"`
	InstallDir string `json:"installDir"`
	ArtURL     string `json:"artUrl"`
}

// Service exposes the registry to the frontend.
type Service struct {
	home string
}

func NewService(home string) *Service { return &Service{home: home} }

// SteamStatus reports whether a directly installed Steam was found.
func (s *Service) SteamStatus() steam.Status {
	_, st := steam.Locate(s.home)
	return st
}

// List returns every supported game with its install state and art URL.
func (s *Service) List() ([]GameInfo, error) {
	st, status := steam.Locate(s.home)
	out := make([]GameInfo, 0, len(registry))
	for _, d := range registry {
		g := GameInfo{ID: d.id, Name: d.name, AppID: d.appID, Loader: d.loader, Available: d.available}
		if status == steam.Found {
			dir, err := st.InstallDir(d.appID)
			if err != nil {
				return nil, err
			}
			g.Installed, g.InstallDir = dir != "", dir
			if st.HeroArt(d.appID) != "" {
				g.ArtURL = artPrefix + d.appID
			}
		}
		out = append(out, g)
	}
	return out, nil
}

// ArtMiddleware serves GET /steam-art/<appid> for registry games only, from Steam's cached hero image.
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

func knownApp(id string) bool {
	if _, err := strconv.ParseUint(id, 10, 32); err != nil {
		return false
	}
	for _, d := range registry {
		if d.appID == id {
			return true
		}
	}
	return false
}
