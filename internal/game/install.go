package game

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/gog"
	"github.com/Rethunk-AI/mortar/internal/lutris"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/steam"
)

const (
	StoreSteam        = "steam"
	StoreFlatpakSteam = "flatpak-steam"
	StoreGOG          = gog.StoreGOG
	StoreGOGHeroic    = gog.StoreHeroic
	StoreLutris       = lutris.StoreLutris
)

// FoundInstall is one discovered game folder and the store it came from.
type FoundInstall struct {
	Store string `json:"store"`
	Dir   string `json:"dir"`
}

var storeOrder = []string{StoreSteam, StoreFlatpakSteam, StoreGOG, StoreGOGHeroic, StoreLutris}

func collect(g Game, home string) []FoundInstall {
	var all []FoundInstall
	seen := map[string]struct{}{}
	add := func(store, dir string) {
		if dir == "" {
			return
		}
		if _, ok := seen[dir]; ok {
			return
		}
		seen[dir] = struct{}{}
		all = append(all, FoundInstall{Store: store, Dir: dir})
	}
	for _, st := range steam.LocateAll(home) {
		dir, err := st.InstallDir(g.SteamAppID())
		if err != nil || dir == "" {
			continue
		}
		store := StoreSteam
		if st.Kind == steam.KindFlatpak {
			store = StoreFlatpakSteam
		}
		add(store, dir)
	}
	for _, in := range gog.Locate(home) {
		add(in.Store, in.Dir)
	}
	for _, in := range lutris.Locate(home) {
		add(StoreLutris, in.Dir)
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
	rank := map[string]int{}
	for i, s := range storeOrder {
		rank[s] = i
	}
	for _, in := range all[1:] {
		if rank[in.Store] < rank[best.Store] {
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
	all = collect(g, home)
	dir, store = pick(all, s.GameFolders[id], s.GameStores[id], g)
	return dir, store, all, nil
}
