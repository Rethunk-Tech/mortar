package gamestore

import (
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gog"
	"github.com/Rethunk-Tech/mortar/internal/lutris"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

const (
	steamKey  = "steam"
	gogKey    = "gog"
	lutrisKey = "lutris"
)

func withCustom(custom []string, rest ...string) []string {
	return append(slices.Clone(custom), rest...)
}

type steamStore struct{}

func (steamStore) Key() string { return steamKey }

func (steamStore) Launchers(goos string) []LauncherSpec {
	steamDir := func(dir string) bool { return fsx.IsDir(filepath.Join(dir, "steamapps")) }
	specs := []LauncherSpec{{
		ID: LauncherSteam, Name: "Steam", Usable: steamDir,
		Looked: func(home string, added map[string][]string) []string {
			return steam.Roots(home, added[LauncherSteam]...)
		},
	}}
	if goos == "linux" {
		specs = append(specs, LauncherSpec{
			ID: LauncherFlatpakSteam, Name: "Steam (Flatpak)", Usable: steamDir,
			Looked: func(home string, _ map[string][]string) []string { return []string{steam.FlatpakRoot(home)} },
		})
	}
	return specs
}

func (steamStore) Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	var out []Install
	for _, st := range steam.LocateAll(home, roots[LauncherSteam]...) {
		dir, err := st.InstallDir(g.SteamAppID())
		if err != nil || dir == "" {
			continue
		}
		store := StoreSteam
		if st.Kind == steam.KindFlatpak {
			store = StoreFlatpakSteam
		}
		out = append(out, Install{Store: store, Dir: dir})
	}
	return out
}

type gogStore struct{}

func (gogStore) Key() string { return gogKey }

func (gogStore) Launchers(goos string) []LauncherSpec {
	specs := []LauncherSpec{{
		ID: LauncherHeroic, Name: "Heroic", Usable: func(dir string) bool { return fsx.IsDir(filepath.Join(dir, "gog_store")) },
		Looked: func(home string, added map[string][]string) []string {
			return gog.HeroicDirs(home, added[LauncherHeroic]...)
		},
	}}
	if goos == "linux" {
		specs = append(specs, LauncherSpec{
			ID: LauncherMinigalaxy, Name: "Minigalaxy", Usable: fsx.IsDir,
			Looked: func(home string, added map[string][]string) []string {
				return withCustom(added[LauncherMinigalaxy], gog.MinigalaxyConfigDirs(home)...)
			},
		})
	}
	galaxy := LauncherSpec{
		ID: LauncherGOG, Name: "GOG", Usable: fsx.IsDir,
		Looked: func(home string, added map[string][]string) []string {
			return gog.OfflineDirs(home, gog.Roots{Games: added[LauncherGOG], Minigalaxy: added[LauncherMinigalaxy]})
		},
	}
	if goos == "windows" {
		galaxy.Name = "GOG Galaxy"
		galaxy.Looked = func(home string, added map[string][]string) []string {
			return append(withCustom(added[LauncherGOG], gog.GalaxyDir()), gog.GamesDirs(home)...)
		}
	}
	return append(specs, galaxy)
}

func (gogStore) Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	var out []Install
	game := gog.Game{ProductID: g.Stores.GOG.ProductID, Folder: g.Stores.GOG.Folder, Marker: g.Marker}
	r := gog.Roots{Heroic: roots[LauncherHeroic], Games: roots[LauncherGOG], Minigalaxy: roots[LauncherMinigalaxy]}
	for _, in := range gog.Locate(home, game, r) {
		out = append(out, Install{Store: in.Store, Dir: in.Dir})
	}
	return out
}

type lutrisStore struct{}

func (lutrisStore) Key() string { return lutrisKey }

func (lutrisStore) Launchers(goos string) []LauncherSpec {
	if goos != "linux" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherLutris, Name: "Lutris", Usable: fsx.IsDir,
		Looked: func(home string, added map[string][]string) []string {
			return lutris.ConfigDirs(home, added[LauncherLutris]...)
		},
	}}
}

func (lutrisStore) Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	var out []Install
	game := lutris.Game{Slug: g.Stores.Lutris.Slug, Keyword: g.Stores.Lutris.Keyword, Marker: g.Marker}
	for _, in := range lutris.Locate(home, game, roots[LauncherLutris]...) {
		out = append(out, Install{Store: StoreLutris, Dir: in.Dir})
	}
	return out
}
