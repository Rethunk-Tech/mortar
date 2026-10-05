// Package settings persists Mortar's user preferences as settings.json.
package settings

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// FileName is the settings file in the data folder.
const FileName = "settings.json"

var (
	accents     = []string{"sand", "moss", "copper", "sky", "rose", "lavender", "teal", "slate"}
	backgrounds = []string{BackgroundImage, BackgroundDesktop, BackgroundSolid}
)

// Played is the last successful launch of a game: the profile id and when it reached Running.
type Played struct {
	Profile string `json:"profile"`
	At      string `json:"at"`
	// GameVersion is the Stardew version from the SMAPI log header of that launch.
	GameVersion string `json:"gameVersion,omitempty"`
	// PlaytimeMs totals the duration of every Mortar-started run of the game that exited cleanly.
	PlaytimeMs int64 `json:"playtimeMs,omitempty"`
}

// The window backgrounds: the chosen wallpaper under the tint, the user's own desktop wallpaper under it, or an opaque colour.
const (
	BackgroundImage   = "image"
	BackgroundDesktop = "desktop"
	BackgroundSolid   = "solid"
)

// Settings is the on-disk shape of settings.json.
type Settings struct {
	FormatVersion int    `json:"formatVersion,omitempty"`
	Language      string `json:"language"`
	Accent        string `json:"accent"`
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
	// Loaders maps a loader id (smapi, bepinex5) to the version Mortar installed. Each loader belongs to one game, so
	// the loader id is a sufficient key.
	Loaders map[string]string `json:"loaders"`
	// Dismissed maps a save folder name to the mod ids whose missing-mod warning the user dismissed for it.
	Dismissed map[string][]string `json:"dismissed"`
	// The signed-in Nexus account, for display only; the API key lives in the keyring. Zero NexusUserID means signed out.
	NexusUserID  int    `json:"nexusUserId"`
	NexusName    string `json:"nexusName"`
	NexusPremium bool   `json:"nexusPremium"`
	// ItchName is the signed-in itch.io account, for display only; the API key lives in the keyring. Empty means signed out.
	ItchName string `json:"itchName"`
	// NxmHandled is whether Mortar is registered for the source link schemes, and NxmPreviousHandlers maps each scheme
	// to the owner Mortar took it from (absent when there was none), which turning the setting off restores. NxmAsked is
	// whether the user has been offered it.
	NxmHandled          bool              `json:"nxmHandled"`
	NxmPreviousHandlers map[string]string `json:"nxmPreviousHandlers"`
	NxmAsked            bool              `json:"nxmAsked"`
	// NxmPreviousName is the display name of the nxm scheme's previous owner when Mortar took over the scheme.
	NxmPreviousName string `json:"nxmPreviousName"`
	// NxmRedirectOtherGames sends nxm:// links for other games to the nxm scheme's previous owner when on.
	NxmRedirectOtherGames *bool `json:"nxmRedirectOtherGames"`
	// ThunderstoreHandleLinks is whether Mortar claims ror2mm:// links from the system. Nil means the source's default (off).
	ThunderstoreHandleLinks *bool `json:"thunderstoreHandleLinks"`
	// NexusPreferredDownloadServer is a seen download_link.json short_name, or empty for Automatic.
	NexusPreferredDownloadServer string `json:"nexusPreferredDownloadServer"`
	// NexusSeenDownloadServers lists short_name values Mortar has seen from Nexus.
	NexusSeenDownloadServers []string `json:"nexusSeenDownloadServers"`
	// AskEndorseMods is whether Mortar may suggest endorsing mods after clean runs. Nil or omitted means on.
	AskEndorseMods *bool `json:"askEndorseMods"`
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
	// SmapiBuilds is whether SMAPI prerelease builds are never offered, shown, or included in updates.
	SmapiBuilds string `json:"smapiBuilds"`
	// LoaderPrefs holds each loader's own settings by loader id; on disk they sit in the loader's block.
	LoaderPrefs map[string]LoaderPrefs `json:"loaderPrefs"`
	// ShowSmapiConsole is whether launches show SMAPI's console. Nil or omitted means on.
	ShowSmapiConsole *bool `json:"showSmapiConsole"`
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
	StartScreen            string `json:"startScreen"`
	Dates                  string `json:"dates"`
	TrashRetentionDays     int    `json:"trashRetentionDays"`
	HistoryEventsKept      int    `json:"historyEventsKept"`
	NotifyDownloadFinished *bool  `json:"notifyDownloadFinished"`
	NotifyDownloadFailed   *bool  `json:"notifyDownloadFailed"`
	NotifyRunCrashed       *bool  `json:"notifyRunCrashed"`
	// DesktopDownloadFinished sends a desktop notification when a download finishes. Nil means off.
	DesktopDownloadFinished *bool `json:"desktopDownloadFinished"`
	// DesktopDownloadFailed sends a desktop notification when a download fails. Nil means on.
	DesktopDownloadFailed *bool `json:"desktopDownloadFailed"`
	// DesktopRunCrashed sends a desktop notification when a Mortar-started run crashes. Nil means on.
	DesktopRunCrashed *bool `json:"desktopRunCrashed"`
	// DesktopModUpdates sends a desktop notification when a background check finds updates. Nil means off.
	DesktopModUpdates        *bool  `json:"desktopModUpdates"`
	Density                  string `json:"density"`
	Theme                    string `json:"theme"`
	GridCardSize             string `json:"gridCardSize"`
	ShowAuthorOnCards        *bool  `json:"showAuthorOnCards"`
	ReduceMotion             string `json:"reduceMotion"`
	ProfileHero              string `json:"profileHero"`
	ReuseFomodChoices        *bool  `json:"reuseFomodChoices"`
	DriftChecks              *bool  `json:"driftChecks"`
	AutoInstallMortarUpdates *bool  `json:"autoInstallMortarUpdates"`
	AutoTrackNexus           bool   `json:"autoTrackNexus"`
	LanName                  string `json:"lanName"`
	LanAutoAcceptPaired      bool   `json:"lanAutoAcceptPaired"`
	DownloadFolder           string `json:"downloadFolder"`
	// WatchFolders are extra folders, joined by the OS path-list separator, that new archives are offered from.
	WatchFolders               string `json:"watchFolders,omitempty"`
	ProfileOrder               string `json:"profileOrder"`
	AutoRetryDownloads         string `json:"autoRetryDownloads"`
	PauseDownloadsWhilePlaying bool   `json:"pauseDownloadsWhilePlaying"`
	SidebarBadges              string `json:"sidebarBadges"`
	ShareIncludeDisabledMods   *bool  `json:"shareIncludeDisabledMods"`
	ShareIncludeFomodChoices   *bool  `json:"shareIncludeFomodChoices"`
	ShareIncludeNotes          *bool  `json:"shareIncludeNotes"`
	ShareIncludeConfigFiles    *bool  `json:"shareIncludeConfigFiles"`
	VerifyNexusMD5             bool   `json:"verifyNexusMD5"`
	// ShowAdultContent lets browse list mods their sites flag as adult.
	ShowAdultContent    bool   `json:"showAdultContent"`
	LaunchAtLogin       bool   `json:"launchAtLogin"`
	StartMinimised      bool   `json:"startMinimised"`
	RememberWindow      bool   `json:"rememberWindow"`
	ExtensionConnection string `json:"extensionConnection"`
	// Games holds per-game prefs.
	Games map[string]*GameSettings `json:"games"`
}

const (
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
	s.LoaderPrefs = map[string]LoaderPrefs{}
	s.Dismissed = map[string][]string{}
	s.NexusSeenDownloadServers = []string{}
	s.LanPort = DefaultLanPort
	s.LanAddresses = []string{}
	s.ListColumns = slices.Clone(defaultListColumns)
	s.ListSortColumn = defaultListSortColumn
	s.ListSortDir = defaultListSortDir
	s.ListGroupBy = defaultListGroupBy
	s.CheckModUpdatesOnStart = on()
	s.TellWhenSmapiOut = on()
	s.SmapiBuilds = SmapiBuildsShow
	s.ShowSmapiConsole = on()
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
	s := &Store{path: filepath.Join(dir, FileName), cur: Defaults()}
	if b, err := fsx.ReadFile(s.path); err == nil {
		loaded, err := decodeFile(b)
		if err != nil {
			corrupt := fmt.Sprintf("%s.corrupt-%d", s.path, time.Now().UnixNano())
			if renameErr := fsx.Rename(s.path, corrupt); renameErr != nil {
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
	if s.cur.LoaderPrefs == nil {
		s.cur.LoaderPrefs = map[string]LoaderPrefs{}
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
	normalizeToggles(&s.cur)
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

// AppendDismissed records token in bucket unless it is already there.
func (s *Store) AppendDismissed(bucket, token string) error {
	_, err := s.Update(func(v *Settings) {
		if slices.Contains(v.Dismissed[bucket], token) {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[bucket] = append(slices.Clone(v.Dismissed[bucket]), token)
		v.Dismissed = next
	})
	return err
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
	next.FormatVersion = datadir.FormatVersion
	doc, err := encodeFile(next)
	if err != nil {
		return s.cur, err
	}
	if err := datadir.WriteVersioned(s.path, doc); err != nil {
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
		case "steam", "flatpak-steam", "gog", "gog-heroic", "gog-minigalaxy", "lutris", "bottles":
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
		cur.LastPlayed[game] = Played{Profile: profile, At: at.UTC().Format(time.RFC3339), GameVersion: ver, PlaytimeMs: prev.PlaytimeMs}
	})
}

// AddPlaytime adds a finished run's duration to the game's total. A game never launched has no entry to add to.
func (s *Store) AddPlaytime(game string, d time.Duration) (Settings, error) {
	if d <= 0 {
		return s.Get(), nil
	}
	return s.Update(func(cur *Settings) {
		if p, ok := cur.LastPlayed[game]; ok {
			p.PlaytimeMs += d.Milliseconds()
			cur.LastPlayed[game] = p
		}
	})
}
