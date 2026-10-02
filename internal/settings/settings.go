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
	// GameVersion is the Stardew version from the SMAPI log header of that launch.
	GameVersion string `json:"gameVersion,omitempty"`
}

// The window backgrounds: the chosen wallpaper under the tint, the user's own desktop wallpaper under it, or an opaque colour.
const (
	BackgroundImage   = "image"
	BackgroundDesktop = "desktop"
	BackgroundSolid   = "solid"
)

// Settings is the on-disk shape of settings.json.
type Settings struct {
	Language string `json:"language"`
	Accent   string `json:"accent"`
	// Background is one of the Background* constants.
	Background string `json:"background"`
	// BackgroundImage is the absolute path of the user's wallpaper; empty means the default one.
	BackgroundImage string `json:"backgroundImage"`
	LastGame        string `json:"lastGame"`
	// LastProfile maps a game id to the id of the profile last open in it.
	LastProfile map[string]string `json:"lastProfile"`
	// LastPlayed maps a game id to the profile last launched to Running and when (RFC3339).
	LastPlayed map[string]Played `json:"lastPlayed"`
	// GameFolders maps a game id to a user-chosen install folder that wins over discovery.
	GameFolders map[string]string `json:"gameFolders"`
	// GameStores maps a game id to the chosen store when several installs were found.
	GameStores map[string]string `json:"gameStores"`
	// LauncherRoots maps a launcher id to folders the user added for it, searched before the usual places.
	LauncherRoots map[string][]string `json:"launcherRoots"`
	// LaunchersConfirmed is whether first run's launcher screen was finished; until then the app opens on it.
	LaunchersConfirmed bool `json:"launchersConfirmed"`
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
	// NxmPreviousName is the display name of NxmPrevious when Mortar took over the scheme.
	NxmPreviousName string `json:"nxmPreviousName"`
	// NxmRedirectOtherGames sends nxm:// links for other games to NxmPrevious when on.
	NxmRedirectOtherGames *bool `json:"nxmRedirectOtherGames"`
	// NexusPreferredDownloadServer is a seen download_link.json short_name, or empty for Automatic.
	NexusPreferredDownloadServer string `json:"nexusPreferredDownloadServer"`
	// NexusSeenDownloadServers lists short_name values Mortar has seen from Nexus.
	NexusSeenDownloadServers []string `json:"nexusSeenDownloadServers"`
	// AskEndorseMods is whether Mortar may suggest endorsing mods after clean runs. Nil or omitted means on.
	AskEndorseMods *bool `json:"askEndorseMods"`
	// BackupsKept is how many save backups to retain, from MinBackupsKept to MaxBackupsKept.
	BackupsKept int `json:"backupsKept"`
	// ListColumns is the Mods list-view columns that are shown. Unknown ids are dropped; an empty list is the default.
	ListColumns []string `json:"listColumns"`
	// ListSortColumn and ListSortDir are the Mods list sort; unknown values become name ascending.
	ListSortColumn string `json:"listSortColumn"`
	ListSortDir    string `json:"listSortDir"`
	// ListGroupBy is how the Mods tab groups the list and grid: none, status, category, source, tag, framework, or author.
	ListGroupBy string `json:"listGroupBy"`
	// CheckModUpdatesOnStart is whether Mortar checks the last-opened profile of each game at startup.
	// Nil or omitted means on.
	CheckModUpdatesOnStart *bool `json:"checkModUpdatesOnStart"`
	// TellWhenSmapiOut is whether Mortar toasts when a newer SMAPI exists. Nil or omitted means on.
	TellWhenSmapiOut *bool `json:"tellWhenSmapiOut"`
	// KeepInTray keeps Mortar in the system tray when the window is closed.
	KeepInTray bool `json:"keepInTray"`
	// LanSharing allows Mortar to discover nearby Mortar users and send or receive profile links.
	LanSharing bool `json:"lanSharing"`
	// LanPort is the LAN sharing HTTP port; zero lets the OS choose one.
	LanPort int `json:"lanPort"`
	// LanAddresses stores the most recently used manual LAN peer addresses.
	LanAddresses []string `json:"lanAddresses"`
	// IncludeBetaReleases offers Mortar prereleases from GitHub when checking for updates.
	IncludeBetaReleases bool `json:"includeBetaReleases"`
	// IncludePrereleaseModVersions offers mod updates whose version has a semver prerelease tag. Default off.
	IncludePrereleaseModVersions bool `json:"includePrereleaseModVersions"`
	// CheckOnlyEnabledMods limits SMAPI update checks to enabled mods when on. Default off.
	CheckOnlyEnabledMods bool `json:"checkOnlyEnabledMods"`
	// EnableModsWhenInstalled is whether new profile entries start with their mods enabled. Nil or omitted means on.
	EnableModsWhenInstalled *bool `json:"enableModsWhenInstalled"`
	// TipsSeen is the empty-state tips the user has dismissed (mods, saves, console, share).
	TipsSeen []string `json:"tipsSeen"`
	// SmapiToastAt is when Mortar last showed the SMAPI-update toast (RFC3339). Empty means never.
	SmapiToastAt string `json:"smapiToastAt"`
	// OverlayEnabled, OverlayPort and OverlayToken are the Mortar SMAPI Bridge stream overlay.
	// OverlayToken is a secret: never log it.
	OverlayEnabled bool   `json:"overlayEnabled"`
	OverlayPort    int    `json:"overlayPort"`
	OverlayToken   string `json:"overlayToken"`
}

const (
	MinBackupsKept  = 1
	MaxBackupsKept  = 50
	DefaultLanPort  = 47630
	MaxLanAddresses = 5
)

// Defaults returns the settings used when no valid file exists.
func on() *bool { v := true; return &v }

func Defaults() Settings {
	return Settings{
		Language: "", Accent: "sand", Background: BackgroundImage, LastProfile: map[string]string{}, LastPlayed: map[string]Played{}, GameFolders: map[string]string{},
		GameStores: map[string]string{}, LauncherRoots: map[string][]string{},
		Loaders: map[string]string{}, Dismissed: map[string][]string{}, NexusSeenDownloadServers: []string{}, LanPort: DefaultLanPort, LanAddresses: []string{}, BackupsKept: backup.DefaultKeep,
		ListColumns: slices.Clone(defaultListColumns), ListSortColumn: defaultListSortColumn, ListSortDir: defaultListSortDir, ListGroupBy: defaultListGroupBy,
		CheckModUpdatesOnStart: on(), TellWhenSmapiOut: on(), EnableModsWhenInstalled: on(), AskEndorseMods: on(),
		LanSharing:  false,
		OverlayPort: DefaultOverlayPort,
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
	if s.cur.GameStores == nil {
		s.cur.GameStores = map[string]string{}
	}
	normalizeStores(&s.cur)
	if s.cur.Loaders == nil {
		s.cur.Loaders = map[string]string{}
	}
	if s.cur.Dismissed == nil {
		s.cur.Dismissed = map[string][]string{}
	}
	if !slices.Contains(accents, s.cur.Accent) {
		s.cur.Accent = Defaults().Accent
	}
	if s.cur.Language != "" && s.cur.Language != "en" {
		s.cur.Language = Defaults().Language
	}
	if !slices.Contains(backgrounds, s.cur.Background) {
		s.cur.Background = Defaults().Background
	}
	if s.cur.BackupsKept < MinBackupsKept || s.cur.BackupsKept > MaxBackupsKept {
		s.cur.BackupsKept = Defaults().BackupsKept
	}
	normalizeToggles(&s.cur)
	normalizeList(&s.cur)
	normalizeTips(&s.cur)
	normalizeOverlay(&s.cur)
	normalizeNexus(&s.cur)
	normalizeLAN(&s.cur)
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
	if next.Language != "" && next.Language != "en" {
		return s.cur, fmt.Errorf("unknown language %q", next.Language)
	}
	if !slices.Contains(backgrounds, next.Background) {
		return s.cur, fmt.Errorf("unknown background %q", next.Background)
	}
	if next.BackupsKept < MinBackupsKept || next.BackupsKept > MaxBackupsKept {
		return s.cur, fmt.Errorf("backups kept must be %d to %d, got %d", MinBackupsKept, MaxBackupsKept, next.BackupsKept)
	}
	if next.LanPort < 0 || next.LanPort > 65535 {
		return s.cur, fmt.Errorf("LAN port must be between 0 and 65535, got %d", next.LanPort)
	}
	next.LastPlayed = validLastPlayed(next.LastPlayed)
	normalizeStores(&next)
	normalizeToggles(&next)
	if err := validateList(next); err != nil {
		return s.cur, err
	}
	if err := validateTips(next); err != nil {
		return s.cur, err
	}
	if err := validateOverlay(next); err != nil {
		return s.cur, err
	}
	if err := validateNexus(next); err != nil {
		return s.cur, err
	}
	normalizeList(&next)
	normalizeTips(&next)
	normalizeNexus(&next)
	normalizeLAN(&next)
	if err := datadir.WriteJSON(s.path, next); err != nil {
		return s.cur, err
	}
	s.cur = next
	return next, nil
}

func normalizeStores(s *Settings) {
	if s.LauncherRoots == nil {
		s.LauncherRoots = map[string][]string{}
	}
	if s.GameStores == nil {
		s.GameStores = map[string]string{}
		return
	}
	for game, store := range s.GameStores {
		switch store {
		case "steam", "flatpak-steam", "gog", "gog-heroic", "gog-minigalaxy", "lutris":
		default:
			delete(s.GameStores, game)
		}
	}
}

func normalizeToggles(s *Settings) {
	if s.CheckModUpdatesOnStart == nil {
		s.CheckModUpdatesOnStart = on()
	}
	if s.TellWhenSmapiOut == nil {
		s.TellWhenSmapiOut = on()
	}
	if s.EnableModsWhenInstalled == nil {
		s.EnableModsWhenInstalled = on()
	}
	if s.AskEndorseMods == nil {
		s.AskEndorseMods = on()
	}
}

func normalizeLAN(s *Settings) {
	if s.LanPort < 0 || s.LanPort > 65535 {
		s.LanPort = DefaultLanPort
	}
	if s.LanAddresses == nil {
		s.LanAddresses = []string{}
	}
	if len(s.LanAddresses) > MaxLanAddresses {
		s.LanAddresses = s.LanAddresses[:MaxLanAddresses]
	}
}

// NewModsEnabled is whether newly installed profile entries should start enabled.
func (s Settings) NewModsEnabled() bool {
	return s.EnableModsWhenInstalled == nil || *s.EnableModsWhenInstalled
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
func (s *Store) RecordLastPlayed(game, profile string, at time.Time, gameVersion string) (Settings, error) {
	if game == "" || profile == "" || at.IsZero() {
		return s.Get(), nil
	}
	return s.Update(func(cur *Settings) {
		if cur.LastPlayed == nil {
			cur.LastPlayed = map[string]Played{}
		}
		prev := cur.LastPlayed[game]
		ver := gameVersion
		if ver == "" {
			ver = prev.GameVersion
		}
		cur.LastPlayed[game] = Played{Profile: profile, At: at.UTC().Format(time.RFC3339), GameVersion: ver}
	})
}
