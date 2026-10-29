package gamestore

import (
	"path/filepath"
	"slices"

	gamelocator "github.com/Rethunk-Tech/go-gamelocator"
	"github.com/Rethunk-Tech/go-gamelocator/ea"
	"github.com/Rethunk-Tech/go-gamelocator/epic"
	"github.com/Rethunk-Tech/go-gamelocator/gog"
	"github.com/Rethunk-Tech/go-gamelocator/heroic"
	"github.com/Rethunk-Tech/go-gamelocator/lutris"
	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/go-gamelocator/xbox"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
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
	steamDir := func(dir string) bool { return safefs.IsDir(filepath.Join(dir, "steamapps")) }
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
		ID: LauncherHeroic, Name: "Heroic", Usable: func(dir string) bool {
			return safefs.IsDir(filepath.Join(dir, "gog_store")) || safefs.IsFile(heroic.EpicList(dir))
		},
		Looked: func(home string, added map[string][]string) []string {
			return heroic.ConfigDirs(home, added[LauncherHeroic]...)
		},
	}}
	if goos == "linux" {
		specs = append(specs, LauncherSpec{
			ID: LauncherMinigalaxy, Name: "Minigalaxy", Usable: safefs.IsDir,
			Looked: func(home string, added map[string][]string) []string {
				return withCustom(added[LauncherMinigalaxy], gog.MinigalaxyConfigDirs(home)...)
			},
		})
	}
	galaxy := LauncherSpec{
		ID: LauncherGOG, Name: "GOG", Usable: safefs.IsDir,
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
	game := gog.Game{ProductID: g.Stores.GOG.ProductID, Folder: g.Stores.GOG.Folder, Marker: g.MarkerPath()}
	r := gog.Roots{Heroic: roots[LauncherHeroic], Games: roots[LauncherGOG], Minigalaxy: roots[LauncherMinigalaxy]}
	for _, in := range gog.Locate(home, game, r) {
		inst := Install{Store: gogStoreID(in.Source), Dir: in.Dir}
		if in.Heroic != "" {
			inst.Prefix, inst.Launcher = heroicPrefix(in.Heroic, in.AppName), "heroic:gog:"+in.AppName
		}
		out = append(out, inst)
	}
	return out
}

// gogStoreID is the store id saved in settings for the launcher a GOG copy came from.
func gogStoreID(s gog.Source) string {
	switch s {
	case gog.SourceGOG:
		return StoreGOG
	case gog.SourceHeroic:
		return StoreGOGHeroic
	case gog.SourceMinigalaxy:
		return StoreMinigalaxy
	}
	return StoreGOG
}

// heroicPrefix is the Wine prefix Heroic keeps for appName, only when the catalog-independent name is plain enough to
// be a file name.
func heroicPrefix(cfg, appName string) string {
	if !components.PlainAppName(appName) {
		return ""
	}
	return heroic.Prefix(cfg, appName)
}

type lutrisStore struct{}

func (lutrisStore) Key() string { return lutrisKey }

func (lutrisStore) Launchers(goos string) []LauncherSpec {
	if goos != "linux" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherLutris, Name: "Lutris", Usable: safefs.IsDir,
		Looked: func(home string, added map[string][]string) []string {
			return lutris.ConfigDirs(home, added[LauncherLutris]...)
		},
	}}
}

func (lutrisStore) Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	var out []Install
	// Lutris runs the Windows build under Wine or, for a game with a native Linux build, that build.
	for _, marker := range []string{g.MarkerPath(), g.LinuxMarker} {
		if marker == "" {
			continue
		}
		game := lutris.Game{Slug: g.Stores.Lutris.Slug, Keyword: g.Stores.Lutris.Keyword, Marker: marker}
		for _, in := range lutris.Locate(home, game, roots[LauncherLutris]...) {
			if !slices.ContainsFunc(out, func(o Install) bool { return o.Dir == in.Dir }) {
				inst := Install{Store: StoreLutris, Dir: in.Dir, Prefix: in.Prefix}
				if components.PlainAppName(in.Slug) {
					inst.Launcher = "lutris:" + in.Slug
				}
				out = append(out, inst)
			}
		}
	}
	return out
}

const eaKey = "ea"

// lookupEARegistry is the registry read; tests replace it so they never touch the real registry.
var lookupEARegistry = ea.RegistryDir

type eaStore struct{}

func (eaStore) Key() string { return eaKey }

func (eaStore) Launchers(goos string) []LauncherSpec {
	if goos != "windows" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherEA, Name: "EA App", Usable: safefs.IsDir,
		Looked: func(_ string, added map[string][]string) []string {
			return ea.Dirs(added[LauncherEA]...)
		},
	}}
}

func (eaStore) Discover(_ string, roots map[string][]string, g components.GameInfo) []Install {
	var out []Install
	for _, dir := range ea.Dirs(roots[LauncherEA]...) {
		if found := gamelocator.GameDir(filepath.Join(dir, g.Stores.EA.Folder), g.MarkerPath()); found != "" {
			out = append(out, Install{Store: StoreEA, Dir: found})
		}
	}
	// A game on another drive is in no library folder; its installer's registry entry names it.
	if g.Stores.EA.Registry != "" {
		if found := gamelocator.GameDir(lookupEARegistry(g.Stores.EA.Registry), g.MarkerPath()); found != "" && !slices.ContainsFunc(out, func(in Install) bool { return safefs.SamePath(in.Dir, found) }) {
			out = append(out, Install{Store: StoreEA, Dir: found})
		}
	}
	return out
}

const epicKey = "epic"

type epicStore struct{}

func (epicStore) Key() string { return epicKey }

func (epicStore) Launchers(goos string) []LauncherSpec {
	if goos != "windows" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherEpic, Name: "Epic Games", Usable: func(dir string) bool { return safefs.IsFile(filepath.Join(dir, epic.ListFile)) },
		Looked: func(_ string, added map[string][]string) []string {
			return epic.ListDirs(added[LauncherEpic]...)
		},
	}}
}

func (epicStore) Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	var out []Install
	add := func(store, heroicCfg string, installs []epic.Install) {
		for _, in := range installs {
			if !slices.Contains(g.Stores.Epic.AppNames, in.AppName) {
				continue
			}
			if found := gamelocator.GameDir(in.Dir, g.MarkerPath()); found != "" {
				inst := Install{Store: store, Dir: found}
				if heroicCfg != "" {
					inst.Prefix, inst.Launcher = heroicPrefix(heroicCfg, in.AppName), "heroic:legendary:"+in.AppName
				}
				out = append(out, inst)
			}
		}
	}
	for _, dir := range epic.ListDirs(roots[LauncherEpic]...) {
		add(StoreEpic, "", epic.Installs(dir))
	}
	for _, dir := range heroic.ConfigDirs(home, roots[LauncherHeroic]...) {
		add(StoreEpicHeroic, dir, heroic.EpicInstalls(dir))
	}
	return out
}

const xboxKey = "xbox"

// appxLocation is the install folder of the Windows app package named pkg, as Gale
// (src-tauri/src/profile/launch/platform.rs) and r2modman
// (src/providers/generic/game/directory_resolver/win/XboxGamePassDirectoryResolver.ts) find an Xbox copy
// (Get-AppxPackage), or "" when it is not installed or not on Windows. A test replaces it.
var appxLocation = func(pkg string) string {
	if !components.PlainPackage(pkg) {
		return ""
	}
	return xbox.PackageRoot(pkg)
}

type xboxStore struct{}

func (xboxStore) Key() string { return xboxKey }

// Launchers lists the Xbox app on Windows; its library folder is C:\XboxGames unless the user added another.
func (xboxStore) Launchers(goos string) []LauncherSpec {
	if goos != "windows" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherXbox, Name: "Xbox", Usable: safefs.IsDir,
		Looked: func(_ string, added map[string][]string) []string {
			return withCustom(added[LauncherXbox], xbox.DefaultLibrary())
		},
	}}
}

// Discover finds the copy the Xbox app installed by its package; a game installed with mods allowed keeps its files
// in that folder's Content subfolder.
func (xboxStore) Discover(_ string, _ map[string][]string, g components.GameInfo) []Install {
	dir := appxLocation(g.Stores.Xbox.Package)
	if dir == "" {
		return nil
	}
	for _, d := range []string{dir, filepath.Join(dir, "Content")} {
		if found := gamelocator.GameDir(d, g.MarkerPath()); found != "" {
			return []Install{{Store: StoreXbox, Dir: found}}
		}
	}
	return nil
}
