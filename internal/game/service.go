package game

import (
	"fmt"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/steam"
)

// GameInfo is one listed game with its install state.
type GameInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AppID      string `json:"appId"`
	Loader     string `json:"loader"`
	Available  bool   `json:"available"`
	Installed  bool   `json:"installed"`
	InstallDir string `json:"installDir"`
	ArtURL     string `json:"artUrl"`
}

// Service exposes the games to the frontend.
type Service struct {
	home  string
	store *settings.Store
}

func NewService(home string, store *settings.Store) *Service {
	return &Service{home: home, store: store}
}

// SteamStatus reports whether a directly installed Steam was found.
func (s *Service) SteamStatus() steam.Status {
	_, st := steam.Locate(s.home)
	return st
}

// List returns every listed game with its install state and art URL.
func (s *Service) List() ([]GameInfo, error) {
	st, status := steam.Locate(s.home)
	var stp *steam.Steam
	if status == steam.Found {
		stp = &st
	}
	folders := s.store.Get().GameFolders
	out := make([]GameInfo, 0, len(games)+len(comingLater))
	for _, g := range games {
		info := GameInfo{ID: g.ID(), Name: g.Name(), AppID: g.SteamAppID(), Loader: g.LoaderName(), Available: true}
		dir, err := g.Discover(folders[g.ID()], stp)
		if err != nil {
			return nil, err
		}
		info.Installed, info.InstallDir = dir != "", dir
		if stp != nil && stp.HeroArt(info.AppID) != "" {
			info.ArtURL = artPrefix + info.AppID
		}
		out = append(out, info)
	}
	for _, c := range comingLater {
		info := GameInfo{ID: c.id, Name: c.name, AppID: c.appID, Loader: c.loader}
		if stp != nil && stp.HeroArt(c.appID) != "" {
			info.ArtURL = artPrefix + c.appID
		}
		out = append(out, info)
	}
	return out, nil
}

// InstallDir returns id's install folder, or "" when the game is not installed.
func InstallDir(home string, folders map[string]string, id string) (string, error) {
	g := Find(id)
	if g == nil {
		return "", fmt.Errorf("unknown game %q", id)
	}
	st, status := steam.Locate(home)
	var stp *steam.Steam
	if status == steam.Found {
		stp = &st
	}
	return g.Discover(folders[id], stp)
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
