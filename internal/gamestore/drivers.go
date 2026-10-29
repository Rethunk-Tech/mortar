package gamestore

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

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
		ID: LauncherHeroic, Name: "Heroic", Usable: func(dir string) bool {
			return fsx.IsDir(filepath.Join(dir, "gog_store")) || fsx.IsFile(heroicEpicList(dir))
		},
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
	game := gog.Game{ProductID: g.Stores.GOG.ProductID, Folder: g.Stores.GOG.Folder, Marker: g.MarkerPath()}
	r := gog.Roots{Heroic: roots[LauncherHeroic], Games: roots[LauncherGOG], Minigalaxy: roots[LauncherMinigalaxy]}
	for _, in := range gog.Locate(home, game, r) {
		inst := Install{Store: in.Store, Dir: in.Dir}
		if in.Heroic != "" {
			inst.Prefix, inst.Launcher = HeroicPrefix(in.Heroic, in.AppName), "heroic:gog:"+in.AppName
		}
		out = append(out, inst)
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

// eaDirs are the folders the EA App installs games into, the user's own first. Only Windows has the EA App; on Linux
// an EA game is found through the Bottles or Lutris driver that holds its Wine prefix.
func eaDirs(goos string, env func(string) string, custom ...string) []string {
	out := slices.Clone(custom)
	if goos != "windows" {
		return out
	}
	for _, v := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if pf := env(v); pf != "" {
			out = append(out, filepath.Join(pf, "EA Games"), filepath.Join(pf, "Electronic Arts"))
		}
	}
	return out
}

// lookupEARegistry is the registry read; tests replace it so they never touch the real registry.
var lookupEARegistry = eaRegistryDir

type eaStore struct{}

func (eaStore) Key() string { return eaKey }

func (eaStore) Launchers(goos string) []LauncherSpec {
	if goos != "windows" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherEA, Name: "EA App", Usable: fsx.IsDir,
		Looked: func(_ string, added map[string][]string) []string {
			return eaDirs(goos, os.Getenv, added[LauncherEA]...)
		},
	}}
}

func (eaStore) Discover(_ string, roots map[string][]string, g components.GameInfo) []Install {
	var out []Install
	for _, dir := range eaDirs(runtime.GOOS, os.Getenv, roots[LauncherEA]...) {
		if found := gog.GameDir(filepath.Join(dir, g.Stores.EA.Folder), g.MarkerPath()); found != "" {
			out = append(out, Install{Store: StoreEA, Dir: found})
		}
	}
	// A game on another drive is in no library folder; its installer's registry entry names it.
	if g.Stores.EA.Registry != "" {
		if found := gog.GameDir(lookupEARegistry(g.Stores.EA.Registry), g.MarkerPath()); found != "" && !slices.ContainsFunc(out, func(in Install) bool { return fsx.SamePath(in.Dir, found) }) {
			out = append(out, Install{Store: StoreEA, Dir: found})
		}
	}
	return out
}

const (
	epicKey = "epic"
	// epicInstalled is the Epic Games Launcher's list of the games it installed and where; r2modman reads it in
	// src/providers/generic/game/directory_resolver/win/EGSDirectoryResolver.ts, Gale in src-tauri/src/profile/launch/platform.rs.
	epicInstalled = "LauncherInstalled.dat"
)

// epicDirs are the folders holding the Epic Games Launcher's install list, the user's own first. Only Windows has the
// launcher.
func epicDirs(goos string, env func(string) string, custom ...string) []string {
	out := slices.Clone(custom)
	if pd := env("ProgramData"); goos == "windows" && pd != "" {
		out = append(out, filepath.Join(pd, "Epic", "UnrealEngineLauncher"))
	}
	return out
}

type epicStore struct{}

func (epicStore) Key() string { return epicKey }

func (epicStore) Launchers(goos string) []LauncherSpec {
	if goos != "windows" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherEpic, Name: "Epic Games", Usable: func(dir string) bool { return fsx.IsFile(filepath.Join(dir, epicInstalled)) },
		Looked: func(_ string, added map[string][]string) []string {
			return epicDirs(goos, os.Getenv, added[LauncherEpic]...)
		},
	}}
}

type epicInstall struct {
	Location string `json:"InstallLocation"`
	AppName  string `json:"AppName"`
}

// epicInstalls reads the install list in dir; an unreadable list holds nothing.
func epicInstalls(dir string) []epicInstall {
	b, err := fsx.ReadFile(filepath.Join(dir, epicInstalled))
	if err != nil {
		return nil
	}
	var list struct {
		Installs []epicInstall `json:"InstallationList"`
	}
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	return list.Installs
}

// heroicEpicList is the list of the Epic games Heroic installed through legendary, below a Heroic config folder.
func heroicEpicList(heroicDir string) string {
	return filepath.Join(heroicDir, "legendaryConfig", "legendary", "installed.json")
}

// heroicEpicInstalls reads the list in heroicDir; an unreadable list holds nothing.
func heroicEpicInstalls(heroicDir string) []epicInstall {
	b, err := fsx.ReadFile(heroicEpicList(heroicDir))
	if err != nil {
		return nil
	}
	var list map[string]struct {
		AppName string `json:"app_name"`
		Path    string `json:"install_path"`
	}
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	out := make([]epicInstall, 0, len(list))
	for _, in := range list {
		out = append(out, epicInstall{Location: in.Path, AppName: in.AppName})
	}
	return out
}

func (epicStore) Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	var out []Install
	add := func(store, heroic string, installs []epicInstall) {
		for _, in := range installs {
			if !slices.Contains(g.Stores.Epic.AppNames, in.AppName) {
				continue
			}
			if found := gog.GameDir(in.Location, g.MarkerPath()); found != "" {
				inst := Install{Store: store, Dir: found}
				if heroic != "" {
					inst.Prefix, inst.Launcher = HeroicPrefix(heroic, in.AppName), "heroic:legendary:"+in.AppName
				}
				out = append(out, inst)
			}
		}
	}
	for _, dir := range epicDirs(runtime.GOOS, os.Getenv, roots[LauncherEpic]...) {
		add(StoreEpic, "", epicInstalls(dir))
	}
	for _, dir := range gog.HeroicDirs(home, roots[LauncherHeroic]...) {
		add(StoreEpicHeroic, dir, heroicEpicInstalls(dir))
	}
	return out
}

// EpicAppName is the AppName the Epic Games Launcher lists for the install in dir (its folder, or the one above its
// game folder), else the first of names: a game sold as several products is started as the one installed.
func EpicAppName(dir string, names []string) string {
	for _, list := range epicDirs(runtime.GOOS, os.Getenv) {
		for _, in := range epicInstalls(list) {
			loc := filepath.Clean(in.Location)
			if slices.Contains(names, in.AppName) && (strings.EqualFold(loc, dir) || strings.EqualFold(loc, filepath.Dir(dir))) {
				return in.AppName
			}
		}
	}
	if len(names) == 0 {
		return ""
	}
	return names[0]
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
	return appxRoot(pkg)
}

type xboxStore struct{}

func (xboxStore) Key() string { return xboxKey }

// Launchers lists the Xbox app on Windows; its library folder is C:\XboxGames unless the user added another.
func (xboxStore) Launchers(goos string) []LauncherSpec {
	if goos != "windows" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherXbox, Name: "Xbox", Usable: fsx.IsDir,
		Looked: func(_ string, added map[string][]string) []string {
			return withCustom(added[LauncherXbox], filepath.Join(cmp.Or(os.Getenv("SystemDrive"), "C:")+`\`, "XboxGames"))
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
		if found := gog.GameDir(d, g.MarkerPath()); found != "" {
			return []Install{{Store: StoreXbox, Dir: found}}
		}
	}
	return nil
}
