// Package gog locates Stardew Valley from GOG Galaxy, the offline installer, and Heroic.
package gog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const (
	// AppID is Stardew Valley's GOG product id.
	AppID = "1453375253"
	// StoreGOG is a GOG Galaxy or offline-installer copy.
	StoreGOG = "gog"
	// StoreHeroic is a copy Heroic installed from GOG.
	StoreHeroic = "gog-heroic"
	marker      = "Stardew Valley.dll"
)

// Install is one GOG-sourced game folder.
type Install struct {
	Dir   string
	Store string
}

// Roots are folders the user added: Heroic config folders and folders of GOG games.
type Roots struct {
	Heroic []string
	Games  []string
}

// Locate finds GOG Stardew folders under home (and Windows Galaxy / default paths), the user's own folders first.
func Locate(home string, r Roots) []Install {
	var out []Install
	seen := map[string]struct{}{}
	add := func(dir, store string) {
		dir = gameDir(dir)
		if dir == "" {
			return
		}
		if _, ok := seen[dir]; ok {
			return
		}
		seen[dir] = struct{}{}
		out = append(out, Install{Dir: dir, Store: store})
	}
	add(galaxyPath(), StoreGOG)
	for _, dir := range GamesDirs(home, r.Games...) {
		add(filepath.Join(dir, "Stardew Valley"), StoreGOG)
	}
	for _, cfg := range HeroicDirs(home, r.Heroic...) {
		for _, dir := range heroicInstalls(cfg) {
			add(dir, StoreHeroic)
		}
	}
	return out
}

// GamesDirs are the folders GOG installers put games in, the user's own first.
func GamesDirs(home string, custom ...string) []string {
	out := slices.Clone(custom)
	if runtime.GOOS == "windows" {
		return append(out, windowsGamesDirs()...)
	}
	return append(out, filepath.Join(home, "GOG Games"))
}

// HeroicDirs are Heroic's config folders (native, then Flatpak), the user's own first.
func HeroicDirs(home string, custom ...string) []string {
	out := slices.Clone(custom)
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			out = append(out, filepath.Join(appData, "heroic"))
		}
		return out
	}
	return append(out,
		filepath.Join(home, ".config", "heroic"),
		filepath.Join(home, ".var", "app", "com.heroicgameslauncher.hgl", "config", "heroic"),
		filepath.Join(home, "snap", "heroic", "current", ".config", "heroic"),
	)
}

func gameDir(dir string) string {
	if hasMarker(dir) {
		return dir
	}
	nested := filepath.Join(dir, "game")
	if hasMarker(nested) {
		return nested
	}
	return ""
}

func hasMarker(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, marker))
	return err == nil && st.Mode().IsRegular()
}

type heroicGame struct {
	AppName     string `json:"appName"`
	InstallPath string `json:"install_path"`
}

func heroicInstalls(cfg string) []string {
	path := filepath.Join(cfg, "gog_store", "installed.json")
	b, err := fsx.ReadFile(path)
	if err != nil {
		return nil
	}
	var games []heroicGame
	var asList struct {
		Games []heroicGame `json:"games"`
	}
	switch {
	case json.Unmarshal(b, &asList) == nil && len(asList.Games) > 0:
		games = asList.Games
	case json.Unmarshal(b, &games) == nil:
	default:
		var keyed map[string]heroicGame
		if json.Unmarshal(b, &keyed) != nil {
			return nil
		}
		for id, g := range keyed {
			if g.AppName == "" {
				g.AppName = id
			}
			games = append(games, g)
		}
	}
	var dirs []string
	for _, g := range games {
		if g.AppName != AppID {
			continue
		}
		dirs = append(dirs, g.InstallPath)
	}
	return dirs
}
