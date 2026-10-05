package game

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/gamestore"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

const (
	StoreSteam        = gamestore.StoreSteam
	StoreFlatpakSteam = gamestore.StoreFlatpakSteam
	StoreGOG          = gamestore.StoreGOG
	StoreGOGHeroic    = gamestore.StoreGOGHeroic
	StoreMinigalaxy   = gamestore.StoreMinigalaxy
	StoreLutris       = gamestore.StoreLutris
)

// FoundInstall is one discovered game folder and the store it came from.
type FoundInstall struct {
	Store string `json:"store"`
	Dir   string `json:"dir"`
}

// Launcher ids, each the source of one or more stores' installs.
const (
	LauncherSteam        = gamestore.LauncherSteam
	LauncherFlatpakSteam = gamestore.LauncherFlatpakSteam
	LauncherHeroic       = gamestore.LauncherHeroic
	LauncherLutris       = gamestore.LauncherLutris
	LauncherGOG          = gamestore.LauncherGOG
	LauncherMinigalaxy   = gamestore.LauncherMinigalaxy
)

// roots are the folders the user added for a launcher.
func roots(s settings.Settings, launcher string) []string { return s.LauncherRoots[launcher] }

func collect(g Game, home string, s settings.Settings) []FoundInstall {
	info, ok := catalogGame(g.ID())
	if !ok {
		return nil
	}
	var all []FoundInstall
	for _, in := range gamestore.Discover(home, s.LauncherRoots, info) {
		all = append(all, FoundInstall{Store: in.Store, Dir: in.Dir})
	}
	return all
}

func pick(all []FoundInstall, override, preferred string, g Game) (dir, store string) {
	if override != "" && g.ValidInstall(override) == nil {
		for _, in := range all {
			if in.Dir == override {
				return override, in.Store
			}
		}
		return override, ""
	}
	if preferred != "" {
		for _, in := range all {
			if in.Store == preferred {
				return in.Dir, in.Store
			}
		}
	}
	if len(all) == 0 {
		return "", ""
	}
	best := all[0]
	for _, in := range all[1:] {
		if gamestore.Rank(in.Store) < gamestore.Rank(best.Store) {
			best = in
		}
	}
	return best.Dir, best.Store
}

// Resolve finds the selected install for id.
func Resolve(home string, s settings.Settings, id string) (dir, store string, all []FoundInstall, err error) {
	g := Find(id)
	if g == nil {
		return "", "", nil, fmt.Errorf("unknown game %q", id)
	}
	all = collect(g, home, s)
	dir, store = pick(all, s.GameFolders[id], s.GameStores[id], g)
	return dir, store, all, nil
}
