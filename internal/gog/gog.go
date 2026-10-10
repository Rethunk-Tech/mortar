// Package gog locates a game installed from GOG: by GOG Galaxy, the offline installer, Heroic or Minigalaxy.
package gog

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

const (
	// StoreGOG is a GOG Galaxy or offline-installer copy.
	StoreGOG = "gog"
	// StoreHeroic is a copy Heroic installed from GOG.
	StoreHeroic = "gog-heroic"
	// StoreMinigalaxy is a copy Minigalaxy installed from GOG.
	StoreMinigalaxy = "gog-minigalaxy"
)

// Game identifies one game to the GOG locators.
type Game struct {
	// ProductID is the game's GOG product id, as Galaxy's registry and Heroic's installed.json name it.
	ProductID string
	// Folder is the folder name GOG installers and Minigalaxy give the game inside a games folder.
	Folder string
	// Marker is a file every install of the game holds, at its root or one "game" folder down.
	Marker string
}

// Install is one GOG-sourced game folder.
type Install struct {
	Dir   string
	Store string
}

// Roots are folders the user added: Heroic config folders and folders of GOG games.
type Roots struct {
	Heroic []string
	Games  []string
	// Minigalaxy are Minigalaxy install folders the user added.
	Minigalaxy []string
}

// Locate finds g's GOG folders under home (and Windows Galaxy / default paths), the user's own folders first.
func Locate(home string, g Game, r Roots) []Install {
	var out []Install
	seen := map[string]struct{}{}
	add := func(dir, store string) {
		dir = GameDir(dir, g.Marker)
		if dir == "" {
			return
		}
		if _, ok := seen[dir]; ok {
			return
		}
		seen[dir] = struct{}{}
		out = append(out, Install{Dir: dir, Store: store})
	}
	add(galaxyPath(g.ProductID), StoreGOG)
	for _, dir := range MinigalaxyInstallDirs(home, r.Minigalaxy...) {
		add(filepath.Join(dir, g.Folder), StoreMinigalaxy)
	}
	for _, dir := range OfflineDirs(home, r) {
		add(filepath.Join(dir, g.Folder), StoreGOG)
	}
	for _, cfg := range HeroicDirs(home, r.Heroic...) {
		for _, dir := range heroicInstalls(cfg, g.ProductID) {
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

// OfflineDirs are the GOG games folders not owned by Minigalaxy, which installs into a folder that is often GOG's
// default too; a folder is credited to one launcher only.
func OfflineDirs(home string, r Roots) []string {
	mini := MinigalaxyInstallDirs(home, r.Minigalaxy...)
	return slices.DeleteFunc(GamesDirs(home, r.Games...), func(d string) bool { return slices.Contains(mini, d) })
}

// MinigalaxyConfigDirs are Minigalaxy's config folders (native, then Flatpak).
func MinigalaxyConfigDirs(home string) []string {
	return []string{
		filepath.Join(home, ".config", "minigalaxy"),
		filepath.Join(home, ".var", "app", "io.github.sharkwouter.Minigalaxy", "config", "minigalaxy"),
	}
}

// MinigalaxyInstallDirs are the folders Minigalaxy installs games into: the user's own first, then each config's
// install_dir (its default, ~/GOG Games, when the config does not set one).
func MinigalaxyInstallDirs(home string, custom ...string) []string {
	out := slices.Clone(custom)
	for _, cfg := range MinigalaxyConfigDirs(home) {
		b, err := fsx.ReadFile(filepath.Join(cfg, "config.json"))
		if err != nil {
			continue
		}
		var c struct {
			InstallDir string `json:"install_dir"`
		}
		if json.Unmarshal(b, &c) != nil || c.InstallDir == "" {
			c.InstallDir = filepath.Join(home, "GOG Games")
		}
		if !slices.Contains(out, c.InstallDir) {
			out = append(out, c.InstallDir)
		}
	}
	return out
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

// GameDir is dir when it holds marker (a slash path below dir, so "Game/Bin/x.exe" works), else its "game" subfolder when that does (GOG's offline layout), else "".
func GameDir(dir, marker string) string {
	if dir == "" {
		return ""
	}
	dir = filepath.Clean(dir)
	for _, d := range []string{dir, filepath.Join(dir, "game")} {
		if st, err := os.Stat(filepath.Join(d, marker)); err == nil && st.Mode().IsRegular() {
			return d
		}
	}
	return ""
}

type heroicGame struct {
	AppName     string `json:"appName"`
	InstallPath string `json:"install_path"`
}

func heroicInstalls(cfg, productID string) []string {
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
			g.AppName = cmp.Or(g.AppName, id)
			games = append(games, g)
		}
	}
	var dirs []string
	for _, g := range games {
		if g.AppName != productID {
			continue
		}
		dirs = append(dirs, g.InstallPath)
	}
	return dirs
}
