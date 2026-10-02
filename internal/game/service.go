package game

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/steam"
)

// GameInfo is one listed game with its install state.
type GameInfo struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	AppID      string         `json:"appId"`
	Loader     string         `json:"loader"`
	Available  bool           `json:"available"`
	Installed  bool           `json:"installed"`
	InstallDir string         `json:"installDir"`
	ArtURL     string         `json:"artUrl"`
	Store      string         `json:"store"`
	Installs   []FoundInstall `json:"installs"`
}

// SteamAccess is whether Flatpak Steam can read Mortar's data folder.
type SteamAccess struct {
	Needed  bool   `json:"needed"`
	Granted bool   `json:"granted"`
	Command string `json:"command"`
}

// Service exposes the games to the frontend.
type Service struct {
	home    string
	store   *settings.Store
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
		return fmt.Errorf("cannot reset %s while it is running", id)
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
	return os.RemoveAll(clean)
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
func (s *Service) List() ([]GameInfo, error) {
	cur := s.store.Get()
	out := make([]GameInfo, 0, len(games)+len(comingLater))
	for _, g := range games {
		info := GameInfo{ID: g.ID(), Name: g.Name(), AppID: g.SteamAppID(), Loader: g.LoaderName(), Available: true, Installs: []FoundInstall{}}
		dir, store, all, err := Resolve(s.home, cur, g.ID())
		if err != nil {
			return nil, err
		}
		info.Installed, info.InstallDir, info.Store = dir != "", dir, store
		if all != nil {
			info.Installs = all
		}
		info.ArtURL = artFor(s.home, info.AppID)
		out = append(out, info)
	}
	for _, c := range comingLater {
		info := GameInfo{ID: c.id, Name: c.name, AppID: c.appID, Loader: c.loader, Installs: []FoundInstall{}}
		info.ArtURL = artFor(s.home, c.appID)
		out = append(out, info)
	}
	return out, nil
}

// InstallDir returns id's install folder, or "" when the game is not installed.
func InstallDir(home string, s settings.Settings, id string) (string, error) {
	dir, _, _, err := Resolve(home, s, id)
	return dir, err
}

// SteamAccess reports the Flatpak Steam filesystem override for Mortar's data folder.
func (s *Service) SteamAccess() (SteamAccess, error) {
	data, err := datadir.Dir()
	if err != nil {
		return SteamAccess{}, err
	}
	acc := SteamAccess{Command: steam.OverrideCommand(data)}
	_, store, _, err := Resolve(s.home, s.store.Get(), "stardew")
	if err != nil {
		return SteamAccess{}, err
	}
	if store != StoreFlatpakSteam {
		return acc, nil
	}
	acc.Needed = true
	show, err := steam.ShowOverride()
	acc.Granted = err == nil && steam.HasFilesystem(show, data)
	return acc, nil
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
