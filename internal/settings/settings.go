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
	// Shortcuts maps action id to a chord such as Ctrl+K. Missing ids use DefaultShortcuts.
	Shortcuts map[string]string `json:"shortcuts"`
	// OnPlay is stay, minimise, or hide (to the tray) when a game launches.
	OnPlay                     string `json:"onPlay"`
	ParallelDownloads          int    `json:"parallelDownloads"`
	UpdateCheckIntervalMinutes int    `json:"updateCheckIntervalMinutes"`
	// NotifyModUpdates toasts when a background check finds updates. Nil means off.
	NotifyModUpdates *bool `json:"notifyModUpdates"`
	// UpdateDigest controls in-app digests after background mod-update checks: off, each, or daily.
	UpdateDigest string `json:"updateDigest"`
	// LastModUpdateDigest is the key+version set last digested (see updatesvc digest keys).
	LastModUpdateDigest []string `json:"lastModUpdateDigest,omitempty"`
	// LastModUpdateDigestAt is when Mortar last showed the mod-update digest toast (RFC3339).
	LastModUpdateDigestAt string `json:"lastModUpdateDigestAt,omitempty"`
	KeepDownloadArchives  bool   `json:"keepDownloadArchives"`
	// StoreRetentionDays is unused store-item lifetime; 0 means keep forever.
	StoreRetentionDays int    `json:"storeRetentionDays"`
	DefaultModsView    string `json:"defaultModsView"`
	// ConfirmRemovals asks before removing mods. Nil means on.
	ConfirmRemovals *bool `json:"confirmRemovals"`
	// BackgroundBadgeChecks fills sidebar badges for other profiles. Nil means on.
	BackgroundBadgeChecks *bool `json:"backgroundBadgeChecks"`
	// StartScreen is last (last opened profile) or gameselect.
	StartScreen                string `json:"startScreen"`
	Dates                      string `json:"dates"`
	TrashRetentionDays         int    `json:"trashRetentionDays"`
	HistoryEventsKept          int    `json:"historyEventsKept"`
	NotifyDownloadFinished     *bool  `json:"notifyDownloadFinished"`
	NotifyDownloadFailed       *bool  `json:"notifyDownloadFailed"`
	NotifyRunCrashed           *bool  `json:"notifyRunCrashed"`
	Density                    string `json:"density"`
	Theme                      string `json:"theme"`
	GridCardSize               string `json:"gridCardSize"`
	ShowAuthorOnCards          *bool  `json:"showAuthorOnCards"`
	ReduceMotion               string `json:"reduceMotion"`
	ProfileHero                string `json:"profileHero"`
	ReuseFomodChoices          *bool  `json:"reuseFomodChoices"`
	DriftChecks                *bool  `json:"driftChecks"`
	AutoInstallMortarUpdates   *bool  `json:"autoInstallMortarUpdates"`
	AutoTrackNexus             bool   `json:"autoTrackNexus"`
	LanName                    string `json:"lanName"`
	LanAutoAcceptSameAccount   bool   `json:"lanAutoAcceptSameAccount"`
	DownloadFolder             string `json:"downloadFolder"`
	ProfileOrder               string `json:"profileOrder"`
	AutoRetryDownloads         string `json:"autoRetryDownloads"`
	PauseDownloadsWhilePlaying bool   `json:"pauseDownloadsWhilePlaying"`
	SidebarBadges              string `json:"sidebarBadges"`
	ShareIncludeDisabledMods   *bool  `json:"shareIncludeDisabledMods"`
	ShareIncludeFomodChoices   *bool  `json:"shareIncludeFomodChoices"`
	ShareIncludeNotes          *bool  `json:"shareIncludeNotes"`
	ShareIncludeConfigFiles    *bool  `json:"shareIncludeConfigFiles"`
	VerifyNexusMD5             bool   `json:"verifyNexusMD5"`
	LaunchAtLogin              bool   `json:"launchAtLogin"`
	StartMinimised             bool   `json:"startMinimised"`
	RememberWindow             bool   `json:"rememberWindow"`
	ExtensionConnection        string `json:"extensionConnection"`
	// Games holds per-game prefs (Stardew Valley today).
	Games map[string]*GameSettings `json:"games"`

	// One-time migration from the pre-registry root fields. Cleared after Open.
	LegacyBackupBeforePlay            string `json:"backupBeforePlay,omitempty"`
	LegacyLaunchBackupsKept           int    `json:"launchBackupsKept,omitempty"`
	LegacyUpdateModsBeforePlayDefault bool   `json:"updateModsBeforePlayDefault,omitempty"`
	LegacyRunsKept                    int    `json:"runsKept,omitempty"`
	LegacyConsoleLogCap               int    `json:"consoleLogCap,omitempty"`
	LegacyNxmDefaultProfile           string `json:"nxmDefaultProfile,omitempty"`
	LegacyCosmeticConflicts           string `json:"cosmeticConflicts,omitempty"`
	LegacyEnableRequirements          string `json:"enableRequirements,omitempty"`
	LegacyMissingRequirements         string `json:"missingRequirements,omitempty"`
	LegacySmapiBuilds                 string `json:"smapiBuilds,omitempty"`
	LegacyDefaultLaunchMethod         string `json:"defaultLaunchMethod,omitempty"`
	LegacyShowSmapiConsole            *bool  `json:"showSmapiConsole,omitempty"`
	LegacyConsoleLevel                string `json:"consoleLevel,omitempty"`
	LegacyConsoleTimestamps           *bool  `json:"consoleTimestamps,omitempty"`
	LegacyConsoleFollow               *bool  `json:"consoleFollow,omitempty"`
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
	s := defaultPrefs()
	s.Language = ""
	s.Accent = "sand"
	s.Background = BackgroundImage
	s.LastProfile = map[string]string{}
	s.LastPlayed = map[string]Played{}
	s.GameFolders = map[string]string{}
	s.GameStores = map[string]string{}
	s.LauncherRoots = map[string][]string{}
	s.Loaders = map[string]string{}
	s.Dismissed = map[string][]string{}
	s.NexusSeenDownloadServers = []string{}
	s.LanPort = DefaultLanPort
	s.LanAddresses = []string{}
	s.BackupsKept = backup.DefaultKeep
	s.ListColumns = slices.Clone(defaultListColumns)
	s.ListSortColumn = defaultListSortColumn
	s.ListSortDir = defaultListSortDir
	s.ListGroupBy = defaultListGroupBy
	s.CheckModUpdatesOnStart = on()
	s.TellWhenSmapiOut = on()
	s.EnableModsWhenInstalled = on()
	s.AskEndorseMods = on()
	s.LanSharing = false
	s.OverlayPort = DefaultOverlayPort
	s.Shortcuts = DefaultShortcuts()
	s.Games = map[string]*GameSettings{}
	return s
}

// Store reads and writes settings.json under the user data folder.
type Store struct {
	mu          sync.Mutex
	path        string
	cur         Settings
	corruptPath string
}

// Open loads settings from the data folder; a corrupt file is preserved beside it and yields defaults.
func Open() (*Store, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, fileName), cur: Defaults()}
	if b, err := os.ReadFile(s.path); err == nil {
		var loaded Settings
		if err := json.Unmarshal(b, &loaded); err != nil {
			corrupt := fmt.Sprintf("%s.corrupt-%d", s.path, time.Now().UnixNano())
			if renameErr := os.Rename(s.path, corrupt); renameErr != nil {
				return nil, fmt.Errorf("preserve corrupt settings: %w", renameErr)
			}
			s.corruptPath = corrupt
		} else {
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
	if s.cur.Games == nil {
		s.cur.Games = map[string]*GameSettings{}
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
	migrateLegacyGameFields(&s.cur)
	normalizePrefs(&s.cur)
	normalizeList(&s.cur)
	normalizeTips(&s.cur)
	normalizeOverlay(&s.cur)
	normalizeNexus(&s.cur)
	normalizeLAN(&s.cur)
	normalizeShortcuts(&s.cur)
	return s, nil
}

// CorruptPath returns the one-time path of settings preserved during Open, if any.
func (s *Store) CorruptPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.corruptPath
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
	if next.Games == nil {
		next.Games = map[string]*GameSettings{}
	}
	if err := validatePrefs(next); err != nil {
		return s.cur, err
	}
	next.LastPlayed = validLastPlayed(next.LastPlayed)
	normalizeStores(&next)
	normalizeToggles(&next)
	normalizePrefs(&next)
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
	if err := rejectUnknownShortcuts(next); err != nil {
		return s.cur, err
	}
	normalizeList(&next)
	normalizeTips(&next)
	normalizeNexus(&next)
	normalizeLAN(&next)
	normalizeShortcuts(&next)
	if err := rejectDuplicateShortcuts(next); err != nil {
		return s.cur, err
	}
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
