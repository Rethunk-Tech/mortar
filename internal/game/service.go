package game

import (
	"fmt"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/settings"
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
	HasSaves bool   `json:"hasSaves"`
	Loader   string `json:"loader"`
	LoaderID string `json:"loaderId"`
	// Loaders are all the loaders the catalog lists for the game, the first being the primary.
	Loaders []LoaderRef `json:"loaders"`
	Sources []string    `json:"sources"`
	// SourceKeys maps a source id to the catalog's key for the game on it (Nexus domain, Thunderstore community); sources without a key are left out.
	SourceKeys map[string]string `json:"sourceKeys"`
	Available  bool              `json:"available"`
	Installed  bool              `json:"installed"`
	InstallDir string            `json:"installDir"`
	ArtURL     string            `json:"artUrl"`
	Store      string            `json:"store"`
	Installs   []Install         `json:"installs"`
}

// LoaderRef names one of a game's loaders.
type LoaderRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Order, Console and Startup say which profile tabs the loader supports: a computed load order, a console on a
	// published companion, and per-mod startup timings.
	Order   bool `json:"order"`
	Console bool `json:"console"`
	Startup bool `json:"startup"`
}

func loaderRef(gameID string, l components.GameLoader) LoaderRef {
	ref := LoaderRef{ID: l.ID, Name: l.Name}
	if d, ok := LoaderOf(gameID, l.ID); ok {
		_, ref.Order = d.(loader.WithOrder)
		_, hasConsole := d.(loader.Console)
		_, hasCompanion := d.(loader.WithCompanion)
		ref.Console = hasConsole && hasCompanion
		_, ref.Startup = d.(loader.StartupTimings)
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

func artFor(home, appID string) string {
	for _, st := range steam.LocateAll(home) {
		if st.HeroArt(appID) != "" {
			return ArtURL(appID)
		}
	}
	return ""
}

// List returns every listed game with its install state and art URL.
func (s *Service) List() ([]GameInfo, error) { return List(s.home, s.store.Get()) }

// List returns every listed game with its install state under settings cur.
func List(home string, cur settings.Settings) ([]GameInfo, error) {
	catalog := Catalog()
	out := make([]GameInfo, 0, len(catalog))
	for _, c := range catalog {
		info := GameInfo{ID: c.ID, Name: c.Name, Deploy: c.Deploy, AppID: c.SteamAppID(), HasSaves: HasSaves(c.ID), Installs: []Install{}, Sources: make([]string, len(c.Sources)), SourceKeys: map[string]string{}}
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
		info.ArtURL = artFor(home, info.AppID)
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
