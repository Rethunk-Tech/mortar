// Package settings persists Mortar's user preferences as settings.json.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backup"
	"github.com/Rethunk-AI/mortar/internal/datadir"
)

const fileName = "settings.json"

var (
	accents     = []string{"sand", "moss", "copper", "sky"}
	backgrounds = []string{BackgroundImage, BackgroundDesktop, BackgroundSolid}
)

// Played is the last successful launch of a game: the profile id and when it reached Running.
type Played struct {
	Profile string `json:"profile"`
	At      string `json:"at"`
}

// The window backgrounds: the chosen wallpaper under the tint, the user's own desktop wallpaper under it, or an opaque colour.
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
	// LastPlayed maps a game id to the profile last launched to Running and when (RFC3339).
	LastPlayed map[string]Played `json:"lastPlayed"`
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
	// BackupsKept is how many save backups to retain, from MinBackupsKept to MaxBackupsKept.
	BackupsKept int `json:"backupsKept"`
	// ListColumns is the Mods list-view columns that are shown. Unknown ids are dropped; an empty list is the default.
	ListColumns []string `json:"listColumns"`
	// ListSortColumn and ListSortDir are the Mods list sort; unknown values become name ascending.
	ListSortColumn string `json:"listSortColumn"`
	ListSortDir    string `json:"listSortDir"`
}

const (
	MinBackupsKept = 1
	MaxBackupsKept = 50
)

// Defaults returns the settings used when no valid file exists.
func Defaults() Settings {
	return Settings{
		Accent: "sand", Background: BackgroundImage, LastProfile: map[string]string{}, LastPlayed: map[string]Played{}, GameFolders: map[string]string{},
		Loaders: map[string]string{}, Dismissed: map[string][]string{}, BackupsKept: backup.DefaultKeep,
		ListColumns: slices.Clone(defaultListColumns), ListSortColumn: defaultListSortColumn, ListSortDir: defaultListSortDir,
	}
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
	s.cur.LastPlayed = validLastPlayed(s.cur.LastPlayed)
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
	if s.cur.BackupsKept < MinBackupsKept || s.cur.BackupsKept > MaxBackupsKept {
		s.cur.BackupsKept = Defaults().BackupsKept
	}
	normalizeList(&s.cur)
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
	if next.BackupsKept < MinBackupsKept || next.BackupsKept > MaxBackupsKept {
		return s.cur, fmt.Errorf("backups kept must be %d to %d, got %d", MinBackupsKept, MaxBackupsKept, next.BackupsKept)
	}
	next.LastPlayed = validLastPlayed(next.LastPlayed)
	if err := validateList(next); err != nil {
		return s.cur, err
	}
	normalizeList(&next)
	if err := datadir.WriteJSON(s.path, next); err != nil {
		return s.cur, err
	}
	s.cur = next
	return next, nil
}

func validLastPlayed(in map[string]Played) map[string]Played {
	out := map[string]Played{}
	for game, p := range in {
		if game == "" || p.Profile == "" || p.At == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339, p.At); err != nil {
			continue
		}
		out[game] = p
	}
	return out
}

// RecordLastPlayed stores that profile as the last successful launch for game.
func (s *Store) RecordLastPlayed(game, profile string, at time.Time) (Settings, error) {
	if game == "" || profile == "" || at.IsZero() {
		return s.Get(), nil
	}
	return s.Update(func(cur *Settings) {
		if cur.LastPlayed == nil {
			cur.LastPlayed = map[string]Played{}
		}
		cur.LastPlayed[game] = Played{Profile: profile, At: at.UTC().Format(time.RFC3339)}
	})
}
