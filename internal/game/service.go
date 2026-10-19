package game

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/customgame"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/shots"
	"github.com/Rethunk-Tech/mortar/internal/steam"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// GameInfo is one listed game with its install state.
type GameInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Deploy is how the profile reaches the game: "redirect" or "profile", whose profiles have a package order.
	Deploy string `json:"deploy"`
	AppID  string `json:"appId"`
	// HasSaves is true when the catalog gives the game a save folder.
	HasSaves bool `json:"hasSaves"`
	// StartupSettings is true when the game has a startup preferences file a profile's game settings are written to.
	StartupSettings bool `json:"startupSettings"`
	// HasOptions is true when the game has an options file a profile keeps its own copy of.
	HasOptions bool `json:"hasOptions"`
	// HasCaches is true when the catalog lists rebuildable cache files Mortar clears after a profile's mod set changes.
	HasCaches bool `json:"hasCaches"`
	// HasScreenshots is true when the game has a screenshots folder of its own or Steam keeps its screenshots.
	HasScreenshots bool   `json:"hasScreenshots"`
	Loader         string `json:"loader"`
	LoaderID       string `json:"loaderId"`
	// Loaders are all the loaders the catalog lists for the game, the first being the primary.
	Loaders []LoaderRef `json:"loaders"`
	Sources []string    `json:"sources"`
	// FindSources are the sites a mod with no recorded source can be looked up on (FindSourceSites); empty hides the button.
	FindSources []string `json:"findSources"`
	// SourceKeys maps a source id to the catalog's key for the game on it (Nexus domain, Thunderstore community); sources without a key are left out.
	SourceKeys map[string]string `json:"sourceKeys"`
	// Graphics is the game's graphics API choices, nil when the catalog offers none.
	Graphics   *components.Graphics `json:"graphics,omitempty"`
	Available  bool                 `json:"available"`
	Installed  bool                 `json:"installed"`
	InstallDir string               `json:"installDir"`
	ArtURL     string               `json:"artUrl"`
	Store      string               `json:"store"`
	Installs   []Install            `json:"installs"`
}

// LoaderRef names one of a game's loaders.
type LoaderRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Order, Console and Startup say which profile tabs the loader supports: a computed load order, a console that
	// follows the loader's log, and per-mod startup timings.
	Order   bool `json:"order"`
	Console bool `json:"console"`
	Startup bool `json:"startup"`
	// Perf is whether the running game's companion measures frame times and each mod's per-frame cost on a measured
	// launch, which Performance › In game shows.
	Perf bool `json:"perf"`
	// Commands is whether the running game takes console commands, which needs the loader's published companion.
	Commands bool `json:"commands"`
	// Overlay is whether the loader's companion feeds the OBS stream overlay, which the Streaming settings need.
	Overlay bool `json:"overlay"`
	// Share is whether the loader's log can be uploaded to smapi.io, whose parser reads only SMAPI's.
	Share bool `json:"share"`
	// Paste is the paste site that sharing the loader's log opens, after copying the log; "" when it has none.
	Paste string `json:"paste"`
	// Assets is whether a content-pack framework for the loader's formats builds an asset map.
	Assets bool `json:"assets"`
	// Frameworks is whether the loader's mods can be content for a framework mod, which the Framework grouping needs.
	Frameworks bool `json:"frameworks"`
	// IntroSkip is whether the loader's companion can skip the game's intro, which the skipIntro setting needs.
	IntroSkip bool `json:"introSkip"`
	// Builtin is whether the loader is part of the game itself, with nothing for Mortar to install.
	Builtin bool `json:"builtin"`
}

func loaderRef(gameID string, l components.GameLoader) LoaderRef {
	ref := LoaderRef{ID: l.ID, Name: l.Name}
	if d, ok := LoaderOf(gameID, l.ID); ok {
		_, ref.Order = d.(loader.WithOrder)
		_, ref.Console = d.(loader.WithLogs)
		ref.Console = ref.Console || HasPath(gameID, PathLog)
		// Commands, startup timing, in-game performance, the overlay and intro skipping all run in the bridge, which
		// Mortar installs only from the game's own bridge component, so a game without one (Cat Mail Co.'s IL2CPP
		// BepInEx, or a BepInEx 5 game whose bridge is not built) offers none of them.
		bridge := l.Companion != "" && components.HasComponent(gameID, l.Companion)
		_, hasConsole := d.(loader.Console)
		_, hasCompanion := d.(loader.WithCompanion)
		// A BepInEx game takes commands only where the bridge knows the game's console; SMAPI always has one.
		ref.Commands = bridge && hasConsole && hasCompanion && (l.Console || !strings.HasPrefix(l.ID, "bepinex"))
		_, ref.Startup = d.(loader.StartupTimings)
		_, ref.Perf = d.(loader.InGamePerf)
		_, ref.Overlay = d.(loader.StreamOverlay)
		_, ref.IntroSkip = d.(loader.IntroSkipper)
		ref.Startup, ref.Perf, ref.Overlay, ref.IntroSkip = bridge && ref.Startup, bridge && ref.Perf, bridge && ref.Overlay, bridge && ref.IntroSkip
		_, ref.Builtin = d.(loader.Builtin)
		ref.Share = loader.SharesLog(d)
		if p, ok := d.(loader.LogPaste); ok {
			ref.Paste = p.PasteSite()
		}
		ref.Assets = framework.IndexesAssets(d.Formats())
		ref.Frameworks = framework.HasFrameworks(d.Formats())
	}
	return ref
}

// SteamAccess is whether Flatpak Steam can read Mortar's data folder.
type SteamAccess struct {
	Needed  bool   `json:"needed"`
	Granted bool   `json:"granted"`
	Command string `json:"command"`
}

// Service exposes the games to the frontend.
type Service struct {
	home  string
	store *settings.Store
	// Running reports whether the game is running; Reset refuses while it is. Nil means never.
	Running func(gameID string) bool
}

func NewService(home string, store *settings.Store) *Service {
	// The games the player added join the catalog before anything lists it.
	if err := customgame.Load(); err != nil {
		log.Printf("custom games: %v", err)
	}
	return &Service{home: home, store: store}
}

// ResetInstall removes only a validated game install, after refusing active games
// and paths that could erase the user's home or filesystem.
func (s *Service) ResetInstall(id string) error {
	if s.Running != nil && s.Running(id) {
		return usererr.Wrap(usererr.Busy, fmt.Errorf("cannot reset %s while it is running", id))
	}
	dir, err := InstallDir(s.home, s.store.Get(), id)
	if err != nil {
		return err
	}
	if dir == "" {
		return fmt.Errorf("game %q is not installed", id)
	}
	if err := ValidateFolder(id, dir); err != nil {
		return err
	}
	clean := filepath.Clean(dir)
	home, err := filepath.Abs(s.home)
	if err != nil {
		return err
	}
	if clean == string(filepath.Separator) || clean == home {
		return fmt.Errorf("refusing to remove protected path %q", clean)
	}
	return fsx.RemoveAll(clean)
}

// SteamStatus reports whether a usable Steam was found.
func (s *Service) SteamStatus() steam.Status {
	_, st := steam.Locate(s.home)
	return st
}

// artFor is where a game's art loads from: ArtMiddleware serves Steam's cached copy, else Steam's public one, so a
// game shows its art before any launcher is found.
func artFor(appID string) string {
	if appID == "" {
		return ""
	}
	return ArtURL(appID)
}

// CustomLoader is a loader a game the player describes can pick.
type CustomLoader struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CustomLoaders are the loaders the Add a game form offers.
func (s *Service) CustomLoaders() []CustomLoader {
	return []CustomLoader{
		{customgame.BepInEx5, "BepInEx 5"},
		{customgame.BepInEx6, "BepInEx 6 (IL2CPP)"},
		{customgame.MelonLoader, "MelonLoader"},
		{customgame.Shimloader, "Shimloader (Unreal Engine, UE4SS)"},
		{customgame.Folder, "None: drop mods in a Mods folder"},
	}
}

// AddCustomGame adds a game missing from the catalog, started by the executable at exe, and returns it with the
// folder to set as its install.
func (s *Service) AddCustomGame(name, exe, loader, community string) (CustomGame, error) {
	e, dir, err := customgame.Add(name, exe, loader, community)
	return CustomGame{ID: e.ID, Name: e.Name, Folder: dir}, err
}

// CustomGame is a game the player added, and the folder it was added from.
type CustomGame struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
}

// RemoveCustomGame forgets a game the player added; the game's folder and profiles are left on disk.
func (s *Service) RemoveCustomGame(id string) error { return customgame.Remove(id) }

// List returns every listed game with its install state and art URL.
func (s *Service) List() ([]GameInfo, error) { return List(s.home, s.store.Get()) }

// List returns every listed game with its install state under settings cur.
func List(home string, cur settings.Settings) ([]GameInfo, error) {
	catalog := Catalog()
	out := make([]GameInfo, 0, len(catalog))
	for _, c := range catalog {
		info := GameInfo{ID: c.ID, Name: c.Name, Deploy: c.Deploy, AppID: c.SteamAppID(), HasSaves: HasSaves(c.ID), StartupSettings: HasStartupSettings(c.ID), HasOptions: HasPath(c.ID, "options"), HasCaches: len(c.Caches) > 0, HasScreenshots: HasScreenshots(c.ID), Installs: []Install{}, Sources: make([]string, len(c.Sources)), SourceKeys: map[string]string{}, FindSources: FindSourceSites(c), Graphics: c.Graphics}
		info.Loader, info.LoaderID = c.Loaders[0].Name, c.Loaders[0].ID
		for _, l := range c.Loaders {
			info.Loaders = append(info.Loaders, loaderRef(c.ID, l))
		}
		for i, src := range c.Sources {
			info.Sources[i] = src.ID
			if src.Key != "" {
				info.SourceKeys[src.ID] = src.Key
			}
		}
		if g := Find(c.ID); g != nil && c.Enabled {
			info.Available = true
			dir, store, all, err := Resolve(home, cur, g.ID())
			if err != nil {
				return nil, err
			}
			info.Installed, info.InstallDir, info.Store = dir != "", dir, store
			if all != nil {
				info.Installs = all
				for i := range info.Installs {
					info.Installs[i].Version = info.Installs[i].readVersion(c)
				}
			}
		}
		info.ArtURL = artFor(info.AppID)
		if info.ArtURL == "" && c.Art != "" && info.Installed {
			info.ArtURL = installArtPrefix + c.ID
		}
		out = append(out, info)
	}
	return out, nil
}

// InstallDir returns id's install folder, or "" when the game is not installed.
func InstallDir(home string, s settings.Settings, id string) (string, error) {
	dir, _, _, err := Resolve(home, s, id)
	return dir, err
}

// SteamAccess reports the Flatpak Steam filesystem override for Mortar's data folder, needed when any game is installed through Flatpak Steam.
func (s *Service) SteamAccess() (SteamAccess, error) {
	data, err := datadir.Dir()
	if err != nil {
		return SteamAccess{}, err
	}
	acc := SteamAccess{Command: steam.OverrideCommand(data)}
	cur := s.store.Get()
	flatpak := false
	for _, id := range Implemented() {
		_, store, _, err := Resolve(s.home, cur, id)
		if err != nil {
			return SteamAccess{}, err
		}
		flatpak = flatpak || store == StoreFlatpakSteam
	}
	if !flatpak {
		return acc, nil
	}
	acc.Needed = true
	show, err := steam.ShowOverride()
	acc.Granted = err == nil && steam.HasFilesystem(show, data)
	return acc, nil
}

// LoaderLaunchLine is the Steam launch-options line that starts the game's loader from installDir.
func (s *Service) LoaderLaunchLine(gameID, installDir string) (string, error) {
	exe, err := requireSteamExe(gameID, installDir)
	if err != nil {
		return "", err
	}
	return launchWith(exe, ""), nil
}

// NeedsLaunchOption reports whether the game's loader must be started through Steam's launch options.
func (s *Service) NeedsLaunchOption(gameID string) (bool, error) {
	return NeedsLaunchOption(gameID)
}

// LaunchOptionsStartLoader reports whether Steam launch options already start the game's loader.
func (s *Service) LaunchOptionsStartLoader(gameID, options string) (bool, error) {
	return StartsLoader(gameID, options)
}

func requireSteamExe(gameID, dir string) (string, error) {
	if _, err := Require(gameID); err != nil {
		return "", err
	}
	exe, ok := steamExe(gameID, dir)
	if !ok {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("game %q has no loader Steam can start", gameID))
	}
	return exe, nil
}

// GrantSteamAccess runs the Flatpak Steam filesystem override after the user confirms.
func (s *Service) GrantSteamAccess() error {
	data, err := datadir.Dir()
	if err != nil {
		return err
	}
	return steam.GrantFilesystem(data)
}

// ValidateFolder reports why dir cannot be id's install folder override; "" (clearing) is always valid.
func ValidateFolder(id, dir string) error {
	g := Find(id)
	if g == nil {
		return fmt.Errorf("unknown game %q", id)
	}
	if dir == "" {
		return nil
	}
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("%q is not an absolute path", dir)
	}
	return g.ValidInstall(dir)
}

// ProtonBuilds names the Proton builds Steam has that the game could run through, best first, when its install is a
// Windows build outside Steam on Linux; otherwise none, since Steam or a bottle chooses.
func (s *Service) ProtonBuilds(gameID string) []string {
	inst, err := ResolveInstall(s.home, s.store.Get(), gameID, "")
	if err != nil || inst.Runtime != runtime.MortarProton {
		return []string{}
	}
	out := []string{}
	for _, b := range runtime.ProtonBuilds(s.home) {
		out = append(out, filepath.Base(b.Dir))
	}
	return out
}

// ScreenshotFile is the file of one of the game's screenshots, named as Screenshots lists it, to copy as a cover.
func (s *Service) ScreenshotFile(gameID, source, name string) (string, error) {
	dir, ok := ScreenshotDirs(s.home, s.store.Get(), gameID)[source]
	if !ok || !shots.IsPicture(name) {
		return "", usererr.New(usererr.Invalid, "no such screenshot")
	}
	return filepath.Join(dir, name), nil
}

// Screenshots lists the game's screenshots, newest first.
func (s *Service) Screenshots(gameID string) (shots.Gallery, error) {
	return shots.List(ScreenshotDirs(s.home, s.store.Get(), gameID))
}

// OpenScreenshotsFolder shows the folder the screenshots of source are in.
func (s *Service) OpenScreenshotsFolder(gameID, source string) error {
	dir, ok := ScreenshotDirs(s.home, s.store.Get(), gameID)[source]
	if !ok {
		return usererr.Wrap(usererr.NotFound, fmt.Errorf("%s has no %s screenshots folder", gameID, source))
	}
	return datadir.Open(dir)
}
