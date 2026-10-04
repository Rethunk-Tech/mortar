package control

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// ShortcutResult is what profile.shortcut returns.
type ShortcutResult struct {
	Path    string `json:"path,omitempty"`
	Removed bool   `json:"removed,omitempty"`
}

// SteamShortcutResult is what profile.steam returns.
type SteamShortcutResult struct {
	Added   bool `json:"added"`
	Already bool `json:"already"`
}

func (s *Services) gameName(id string) (string, error) {
	list, err := s.Games.List()
	if err != nil {
		return "", err
	}
	for _, g := range list {
		if g.ID == id {
			return g.Name, nil
		}
	}
	return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", id))
}

func (s *Services) profileShortcut(gameID, profileID string, remove bool) (ShortcutResult, error) {
	if s.Plays == nil {
		return ShortcutResult{}, fmt.Errorf("shortcuts are unavailable")
	}
	prof, err := s.resolve(gameID, profileID)
	if err != nil {
		return ShortcutResult{}, err
	}
	gameName, err := s.gameName(gameID)
	if err != nil {
		return ShortcutResult{}, err
	}
	if remove {
		if err := s.Plays.Remove(gameID, prof.ID); err != nil {
			return ShortcutResult{}, err
		}
		return ShortcutResult{Removed: true}, nil
	}
	path, err := s.Plays.Create(gameID, gameName, prof.ID, prof.Name)
	if err != nil {
		return ShortcutResult{}, err
	}
	return ShortcutResult{Path: path}, nil
}

func (s *Services) profileSteam(gameID, profileID string) (SteamShortcutResult, error) {
	if s.Plays == nil {
		return SteamShortcutResult{}, fmt.Errorf("shortcuts are unavailable")
	}
	prof, err := s.resolve(gameID, profileID)
	if err != nil {
		return SteamShortcutResult{}, err
	}
	gameName, err := s.gameName(gameID)
	if err != nil {
		return SteamShortcutResult{}, err
	}
	added, err := s.Plays.AddToSteam(gameID, gameName, prof.ID, prof.Name)
	if err != nil {
		return SteamShortcutResult{}, err
	}
	if added {
		return SteamShortcutResult{Added: true}, nil
	}
	return SteamShortcutResult{Already: true}, nil
}

// SteamLaunchOptionResult is the current or updated Steam launch options line.
type SteamLaunchOptionResult struct {
	Options string `json:"options"`
	Set     bool   `json:"set,omitempty"`
	Cleared bool   `json:"cleared,omitempty"`
}

func (s *Services) gameSteamLaunchOption(gameID string, set, unset bool) (SteamLaunchOptionResult, error) {
	if set && unset {
		return SteamLaunchOptionResult{}, fmt.Errorf("use either --set or --clear, not both")
	}
	if set {
		opts, err := s.Games.SetLaunchOption(gameID)
		if err != nil {
			return SteamLaunchOptionResult{}, err
		}
		return SteamLaunchOptionResult{Options: opts, Set: true}, nil
	}
	if unset {
		opts, err := s.Games.ClearLaunchOption(gameID)
		if err != nil {
			return SteamLaunchOptionResult{}, err
		}
		return SteamLaunchOptionResult{Options: opts, Cleared: true}, nil
	}
	opts, err := s.Games.LaunchOptions(gameID)
	if err != nil {
		return SteamLaunchOptionResult{}, err
	}
	return SteamLaunchOptionResult{Options: opts}, nil
}
