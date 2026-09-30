// Package settings persists Mortar's user preferences as settings.json.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

const fileName = "settings.json"

var (
	accents     = []string{"sand", "moss", "copper", "sky"}
	backgrounds = []string{BackgroundImage, BackgroundDesktop, BackgroundSolid}
)

// The window backgrounds: a wallpaper under the tint, the desktop showing through, or an opaque colour.
const (
	BackgroundImage   = "image"
	BackgroundDesktop = "desktop"
	BackgroundSolid   = "solid"
)

// Settings is the on-disk shape of settings.json.
type Settings struct {
	Accent string `json:"accent"`
	// Background is one of the Background* constants.
	Background string `json:"background"`
	// BackgroundImage is the absolute path of the user's wallpaper; empty means the default one.
	BackgroundImage string `json:"backgroundImage"`
	LastGame        string `json:"lastGame"`
	// LastProfile maps a game id to the id of the profile last open in it.
	LastProfile map[string]string `json:"lastProfile"`
	// GameFolders maps a game id to a user-chosen install folder that wins over Steam discovery.
	GameFolders map[string]string `json:"gameFolders"`
	// Loaders maps a game id to the loader version Mortar installed.
	Loaders map[string]string `json:"loaders"`
	// Dismissed maps a save folder name to the UniqueIDs whose missing-mod warning the user dismissed for it.
	Dismissed map[string][]string `json:"dismissed"`
	// The signed-in Nexus account, for display only; the API key lives in the keyring. Zero NexusUserID means signed out.
	NexusUserID  int    `json:"nexusUserId"`
	NexusName    string `json:"nexusName"`
	NexusPremium bool   `json:"nexusPremium"`
	// NxmHandled is whether Mortar is registered for nxm:// links, and NxmPrevious the owner it took them from (empty
	// when there was none), which turning the setting off restores. NxmAsked is whether the user has been offered it.
	NxmHandled  bool   `json:"nxmHandled"`
	NxmPrevious string `json:"nxmPrevious"`
	NxmAsked    bool   `json:"nxmAsked"`
}

// Defaults returns the settings used when no valid file exists.
func Defaults() Settings {
	return Settings{Accent: "sand", Background: BackgroundImage, LastProfile: map[string]string{}, GameFolders: map[string]string{}, Loaders: map[string]string{}, Dismissed: map[string][]string{}}
}

// Store reads and writes settings.json under the user data folder.
type Store struct {
	mu   sync.Mutex
	path string
	cur  Settings
}

// Open loads settings from the data folder; a missing or corrupt file yields defaults.
func Open() (*Store, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, fileName), cur: Defaults()}
	if b, err := os.ReadFile(s.path); err == nil {
		var loaded Settings
		if json.Unmarshal(b, &loaded) == nil {
			s.cur = loaded
		}
	}
	if s.cur.LastProfile == nil {
		s.cur.LastProfile = map[string]string{}
	}
	if s.cur.GameFolders == nil {
		s.cur.GameFolders = map[string]string{}
	}
	if s.cur.Loaders == nil {
		s.cur.Loaders = map[string]string{}
	}
	if s.cur.Dismissed == nil {
		s.cur.Dismissed = map[string][]string{}
	}
	if !slices.Contains(accents, s.cur.Accent) {
		s.cur.Accent = Defaults().Accent
	}
	if !slices.Contains(backgrounds, s.cur.Background) {
		s.cur.Background = Defaults().Background
	}
	return s, nil
}

// Get returns the current settings.
func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Update applies fn to a copy of the settings, validates, persists atomically and returns the result.
func (s *Store) Update(fn func(*Settings)) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.cur
	fn(&next)
	if !slices.Contains(accents, next.Accent) {
		return s.cur, fmt.Errorf("unknown accent %q", next.Accent)
	}
	if !slices.Contains(backgrounds, next.Background) {
		return s.cur, fmt.Errorf("unknown background %q", next.Background)
	}
	if err := datadir.WriteJSON(s.path, next); err != nil {
		return s.cur, err
	}
	s.cur = next
	return next, nil
}
