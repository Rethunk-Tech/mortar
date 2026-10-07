package game

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	goruntime "runtime"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/steam"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

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
	StoreBottles      = gamestore.StoreBottles
)

// Install origins.
const (
	OriginDiscovered = "discovered"
	// OriginFolder is a folder the user chose that no store reported.
	OriginFolder = "folder"
)

// Install is one game folder: where it came from and how it runs.
type Install struct {
	// ID is stable for a store and folder.
	ID    string `json:"id"`
	Game  string `json:"game"`
	Store string `json:"store"`
	Dir   string `json:"dir"`
	// Prefix is the Wine prefix (a Bottles bottle) a Windows build runs in; empty for other installs.
	Prefix string `json:"prefix,omitempty"`
	// Runtime is the id of the runtime that runs the install: native, proton or wine-prefix.
	Runtime string `json:"runtime"`
	// Platform is the OS the build is for: windows, linux or darwin.
	Platform string `json:"platform"`
	// RuntimeVersion is the compatibility tool Steam runs a Proton install with; empty for a native one.
	RuntimeVersion string `json:"runtimeVersion,omitempty"`
	Origin         string `json:"origin"`
	// Version is the game's version: the loader's reading where it has one, else Steam's build id, an opaque version.
	// It is filled in by List only.
	Version string `json:"version,omitempty"`
}

func newInstall(info components.GameInfo, store, dir, prefix, origin string) Install {
	sum := sha256.Sum256([]byte(store + "\x00" + dir))
	in := Install{ID: hex.EncodeToString(sum[:6]), Game: info.ID, Store: store, Dir: dir, Prefix: prefix, Origin: origin}
	in.Platform = platformOf(info, dir)
	in.Runtime = runtime.IDOf(in.runtime(info, ""))
	in.RuntimeVersion = runtime.Version(in.runtime(info, ""))
	return in
}

// platformOf is the OS the build in dir is for: the native Linux build when dir holds the catalog's LinuxMarker and not
// its Marker, else the platform Marker implies.
func platformOf(info components.GameInfo, dir string) string {
	if info.LinuxMarker != "" && goruntime.GOOS == "linux" && !fsx.IsFile(filepath.Join(dir, info.Marker)) && fsx.IsFile(filepath.Join(dir, info.LinuxMarker)) {
		return "linux"
	}
	return runtime.PlatformOf(info.Marker, goruntime.GOOS)
}

func (in Install) runtime(info components.GameInfo, home string) runtime.Install {
	return runtime.Install{Store: in.Store, Dir: in.Dir, AppID: info.SteamAppID(), Platform: in.Platform, Prefix: in.Prefix, Home: home}
}

// readVersion is the game's version in the install, "" when nothing names one.
func (in Install) readVersion(info components.GameInfo) string {
	if l, ok := PrimaryLoader(in.Game); ok {
		if r, ok := l.(loader.InstalledVersion); ok {
			if v := r.InstalledVersion(in.Dir); v != "" {
				return v
			}
		}
	}
	if (in.Store == gamestore.LauncherSteam || in.Store == gamestore.LauncherFlatpakSteam) && info.SteamAppID() != "" {
		return steam.BuildID(in.Dir, info.SteamAppID())
	}
	return ""
}

// CompatData is the Steam compatdata folder of a Proton install, the parent of its Wine prefix.
func (in Install) CompatData(home string) (string, bool) {
	info, ok := catalogGame(in.Game)
	if !ok {
		return "", false
	}
	return runtime.CompatDataDir(in.runtime(info, home))
}

// Launcher ids, each the source of one or more stores' installs.
const (
	LauncherSteam        = gamestore.LauncherSteam
	LauncherFlatpakSteam = gamestore.LauncherFlatpakSteam
	LauncherHeroic       = gamestore.LauncherHeroic
	LauncherLutris       = gamestore.LauncherLutris
	LauncherGOG          = gamestore.LauncherGOG
	LauncherMinigalaxy   = gamestore.LauncherMinigalaxy
	LauncherBottles      = gamestore.LauncherBottles
)

// roots are the folders the user added for a launcher.
func roots(s settings.Settings, launcher string) []string { return s.LauncherRoots[launcher] }

func collect(g Game, home string, s settings.Settings) []Install {
	info, ok := catalogGame(g.ID())
	if !ok {
		return nil
	}
	var all []Install
	for _, in := range gamestore.Discover(home, s.LauncherRoots, info) {
		all = append(all, newInstall(info, in.Store, in.Dir, in.Prefix, OriginDiscovered))
	}
	return all
}

func pick(all []Install, override, preferred string, g Game) (dir, store string) {
	if override != "" {
		// A hand-edited settings file can hold a folder with a trailing separator.
		override = filepath.Clean(override)
	}
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
func Resolve(home string, s settings.Settings, id string) (dir, store string, all []Install, err error) {
	g := Find(id)
	if g == nil {
		return "", "", nil, fmt.Errorf("unknown game %q", id)
	}
	all = collect(g, home, s)
	dir, store = pick(all, s.GameFolders[id], s.GameStores[id], g)
	return dir, store, all, nil
}

// ResolveInstall returns the install with id pin, or the selected install when pin is empty. A game that is not
// installed yields an Install with no folder.
func ResolveInstall(home string, s settings.Settings, id, pin string) (Install, error) {
	dir, store, all, err := Resolve(home, s, id)
	if err != nil {
		return Install{}, err
	}
	if pin != "" {
		for _, in := range all {
			if in.ID == pin {
				return in, nil
			}
		}
		return Install{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("install %q of %s is no longer found", pin, id))
	}
	for _, in := range all {
		if in.Dir == dir {
			return in, nil
		}
	}
	info, _ := catalogGame(id)
	if dir == "" {
		in := newInstall(info, "", "", "", OriginFolder)
		in.ID = ""
		return in, nil
	}
	// A folder the user chose inside a bottle still runs in it.
	prefix := gamestore.BottleOf(dir)
	if prefix != "" {
		store = StoreBottles
	}
	return newInstall(info, store, dir, prefix, OriginFolder), nil
}

// RunsInBottle reports whether the install runs through a Bottles bottle.
func (in Install) RunsInBottle() bool { return in.Runtime == runtime.WinePrefix }

// BottleCommand is the host command that runs exe with args inside the install's bottle.
func (in Install) BottleCommand(exe string, args ...string) ([]string, error) {
	info, _ := catalogGame(in.Game)
	return runtime.Command(in.runtime(info, ""), exe, args...)
}

// RunInBottle runs exe with args inside the install's bottle through run.
func (in Install) RunInBottle(ctx context.Context, run runtime.Runner, exe string, args ...string) error {
	info, _ := catalogGame(in.Game)
	return runtime.Run(ctx, run, in.runtime(info, ""), exe, args...)
}
