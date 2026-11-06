package settings

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	OnPlayStay     = "stay"
	OnPlayMinimise = "minimise"
	OnPlayHide     = "hide"

	BackupBeforePlayChanged = "changed"
	BackupBeforePlayAlways  = "always"
	BackupBeforePlayNever   = "never"

	CosmeticCollapsed = "collapsed"
	CosmeticExpanded  = "expanded"
	CosmeticHidden    = "hidden"

	AntivirusAutomatic    = "automatic"
	AntivirusClamd        = "clamd"
	AntivirusCommand      = "command"
	AntivirusOff          = "off"
	StartScreenLast       = "last"
	StartScreenGameSelect = "gameselect"

	DatesRelative = "relative"
	DatesAbsolute = "absolute"

	// How WebKit renders on Linux: automatic keeps Wails' choice (the DMA-BUF renderer off where NVIDIA renders),
	// smooth keeps the DMA-BUF renderer and turns compositing off instead, which scrolls at a third of the CPU,
	// compatible turns the DMA-BUF renderer off on any GPU, for a blank or white window.
	GraphicsAutomatic  = "automatic"
	GraphicsSmooth     = "smooth"
	GraphicsCompatible = "compatible"

	ModsViewGrid = "grid"
	ModsViewList = "list"
	// How the Mods list shows implicit mods: in place, beneath the mods that require them, or not at all.
	ImplicitModsShown  = "shown"
	ImplicitModsNested = "nested"
	ImplicitModsHidden = "hidden"

	DensityComfortable = "comfortable"
	DensityCompact     = "compact"
	ThemeDark          = "dark"
	ThemeLight         = "light"
	ThemeSystem        = "system"

	GridCardSmall  = "small"
	GridCardMedium = "medium"
	GridCardLarge  = "large"

	ReduceMotionSystem = "system"
	ReduceMotionAlways = "always"
	ReduceMotionNever  = "never"

	HeroFull    = "full"
	HeroCompact = "compact"
	HeroHidden  = "hidden"

	EnableReqAlways = "always"
	EnableReqAsk    = "ask"
	EnableReqNever  = "never"

	MissingReqAsk          = "ask"
	MissingReqAutodownload = "autodownload"
	MissingReqNever        = "never"

	SmapiBuildsNever   = "never"
	SmapiBuildsShow    = "show"
	SmapiBuildsInclude = "include"

	LaunchSteam  = "steam"
	LaunchDirect = "direct"

	StartFirstOff     = "off"
	StartFirstSteamVR = "steamvr"
	StartFirstALVR    = "alvr"

	ConsoleLevelTrace = "trace"
	ConsoleLevelDebug = "debug"
	ConsoleLevelInfo  = "info"
	ConsoleLevelWarn  = "warn"
	ConsoleLevelError = "error"

	ProfileOrderManual     = "manual"
	ProfileOrderName       = "name"
	ProfileOrderLastPlayed = "lastPlayed"

	ThunderstoreHostAuto = "auto"
	ThunderstoreHostC    = "ccdn.thunderstore.io"
	ThunderstoreHostG    = "gcdn.thunderstore.io"
	ThunderstoreHostH    = "hcdn-1.hcdn.thunderstore.io"

	AutoRetryOff = "off"
	AutoRetry1   = "1"
	AutoRetry3   = "3"

	SidebarBadgesAll      = "problemsAndUpdates"
	SidebarBadgesProblems = "problems"
	SidebarBadgesOff      = "off"

	ConflictScanFull       = "full"
	ConflictScanSkipImages = "skipImages"

	ExtensionAllow = "allow"
	ExtensionOff   = "off"

	GameSettingsEdit = "edit"
	GameSettingsWarn = "warn"

	CacheClearingAuto = "auto"
	CacheClearingOff  = "off"

	SyncFollow = "follow"
	SyncOwn    = "own"
	SyncOff    = "off"

	DependencyNewest = "newest"
	DependencyExact  = "exact"

	OldFilesAsk    = "ask"
	OldFilesDelete = "delete"
	OldFilesKeep   = "keep"

	DefaultRunsKept                   = 20
	DefaultConsoleLogCap              = 20000
	DefaultParallelDownloads          = 3
	DefaultUpdateCheckIntervalMinutes = 180
	DefaultStoreRetentionDays         = 30
	DefaultTrashRetentionDays         = 30
	DefaultHistoryEventsKept          = 200
	DefaultSaveBackupsKept            = 5
	DefaultSaveBackupKeep             = 5
	DefaultInstances                  = 1
	DefaultDelaySeconds               = 5

	MinInstances    = 1
	MaxInstances    = 8
	MinDelaySeconds = 1
	MaxDelaySeconds = 60

	MinRunsKept                   = 1
	MaxRunsKept                   = 100
	MinConsoleLogCap              = 1000
	MaxConsoleLogCap              = 100000
	MinParallelDownloads          = 1
	MinUIScale                    = 80
	MaxUIScale                    = 150
	DefaultUIScale                = 100
	MaxParallelDownloads          = 8
	MinDownloadSpeedLimit         = 0
	MaxDownloadSpeedLimit         = 10000
	MinUpdateCheckIntervalMinutes = 15
	MaxUpdateCheckIntervalMinutes = 24 * 60
	MinStoreRetentionDays         = 0
	MaxStoreRetentionDays         = 3650
	MinTrashRetentionDays         = 1
	MaxTrashRetentionDays         = 365
	MinHistoryEventsKept          = 20
	MaxHistoryEventsKept          = 2000
	MinSaveBackupsKept            = 1
	MaxSaveBackupsKept            = 50
	// SaveBackupHours 0 is off and 24 is daily.
	MinSaveBackupHours = 0
	MaxSaveBackupHours = 7 * 24
)

func off() *bool { v := false; return &v }

func defaultPrefs() Settings {
	return Settings{
		OnPlay:                     OnPlayStay,
		ParallelDownloads:          DefaultParallelDownloads,
		UpdateCheckIntervalMinutes: DefaultUpdateCheckIntervalMinutes,
		NotifyModUpdates:           on(),
		UpdateDigest:               UpdateDigestDaily,
		KeepDownloadArchives:       false,
		StoreRetentionDays:         DefaultStoreRetentionDays,
		DefaultModsView:            ModsViewGrid,
		ImplicitModsView:           ImplicitModsNested,
		ConfirmRemovals:            on(),
		BackgroundBadgeChecks:      on(),
		StartScreen:                StartScreenLast,
		Dates:                      DatesRelative,
		Graphics:                   GraphicsAutomatic,
		TrashRetentionDays:         DefaultTrashRetentionDays,
		HistoryEventsKept:          DefaultHistoryEventsKept,
		Antivirus:                  AntivirusAutomatic,
		NotifyDownloadFinished:     off(),
		NotifyDownloadFailed:       on(),
		NotifyRunCrashed:           on(),
		Density:                    DensityComfortable,
		UIScale:                    DefaultUIScale,
		Theme:                      ThemeDark,
		GridCardSize:               GridCardMedium,
		ShowAuthorOnCards:          on(),
		MarkLocalMods:              on(),
		BrowseCardStats:            defaultBrowseCardStats,
		ReduceMotion:               ReduceMotionSystem,
		ProfileHero:                HeroFull,
		ReuseFomodChoices:          on(),
		DriftChecks:                on(),
		AutoInstallMortarUpdates:   on(),
		AutoTrackNexus:             false,
		LanName:                    "",
		LanAutoAcceptPaired:        false,
		DownloadFolder:             "",
		ProfileOrder:               ProfileOrderManual,
		AutoRetryDownloads:         AutoRetryOff,
		ThunderstoreHost:           ThunderstoreHostAuto,
		SidebarBadges:              SidebarBadgesAll,
		ShareIncludeDisabledMods:   on(),
		ShareIncludeFomodChoices:   on(),
		ShareIncludeNotes:          on(),
		ShareIncludeConfigFiles:    on(),
		ShareIncludeProblemChoices: on(),
		ShareIncludeLibraries:      on(),
		VerifyNexusMD5:             true,
		ShowAdultContent:           false,
		LaunchAtLogin:              false,
		StartMinimised:             false,
		RememberWindow:             true,
		KeepInTray:                 true,
		IncludeBetaReleases:        true,
		LanSharing:                 true,
		LanPort:                    0,
		LanAllowAnyAddress:         false,
		AskEndorseMods:             off(),
		DesktopDownloadFinished:    off(),
		DesktopDownloadFailed:      on(),
		DesktopRunCrashed:          on(),
		DesktopModUpdates:          off(),
		CheckModUpdatesOnStart:     on(),
		ExtensionConnection:        ExtensionAllow,
		Games:                      map[string]*GameSettings{},
	}
}

func normalizePrefs(s *Settings) {
	for _, p := range normStrings(s) {
		if !slices.Contains(p.values, *p.field) {
			*p.field = p.def
		}
	}
	for _, p := range normInts(s) {
		if *p.field < p.lo || *p.field > p.hi {
			*p.field = p.def
		}
	}
	for _, p := range normBools(s) {
		if *p.field == nil {
			*p.field = p.def()
		}
	}
	normalizeUpdateDigest(s)
	moveAutoInstallFolders(s)
	s.BrowseCardStats = sanitizeBrowseCardStats(s.BrowseCardStats)
	if s.Games == nil {
		s.Games = map[string]*GameSettings{}
	}
	for id, g := range s.Games {
		if g == nil {
			d := defaultGameSettingsFor(id)
			s.Games[id] = &d
			continue
		}
		normalizeGame(id, g)
	}
}

// normString is an enumerated setting that falls back to def when its value is not one of values.
type normString struct {
	field  *string
	values []string
	def    string
}

// normInt is a ranged setting that falls back to def outside lo to hi.
type normInt struct {
	field       *int
	lo, hi, def int
}

// normBool is an optional switch that takes def() while unset.
type normBool struct {
	field **bool
	def   func() *bool
}

func normStrings(s *Settings) []normString {
	return []normString{
		{&s.OnPlay, onPlayValues, OnPlayStay},
		{&s.DefaultModsView, modsViewValues, ModsViewGrid},
		{&s.ImplicitModsView, implicitModsValues, ImplicitModsNested},
		{&s.StartScreen, startScreenValues, StartScreenLast},
		{&s.Antivirus, antivirusValues, AntivirusAutomatic},
		{&s.Dates, datesValues, DatesRelative},
		{&s.Graphics, graphicsValues, GraphicsAutomatic},
		{&s.Density, densityValues, DensityComfortable},
		{&s.Theme, themeValues, ThemeDark},
		{&s.GridCardSize, gridCardValues, GridCardMedium},
		{&s.ReduceMotion, reduceMotionValues, ReduceMotionSystem},
		{&s.ProfileHero, heroValues, HeroFull},
		{&s.ProfileOrder, profileOrderValues, ProfileOrderManual},
		{&s.ThunderstoreHost, thunderstoreHostValues, ThunderstoreHostAuto},
		{&s.AutoRetryDownloads, autoRetryValues, AutoRetryOff},
		{&s.SidebarBadges, sidebarBadgesValues, SidebarBadgesAll},
		{&s.ExtensionConnection, extensionConnectionValues, ExtensionAllow},
		{&s.SmapiBuilds, smapiBuildsValues, SmapiBuildsShow},
	}
}

func normInts(s *Settings) []normInt {
	return []normInt{
		{&s.ParallelDownloads, MinParallelDownloads, MaxParallelDownloads, DefaultParallelDownloads},
		{&s.DownloadSpeedLimit, MinDownloadSpeedLimit, MaxDownloadSpeedLimit, 0},
		{&s.UpdateCheckIntervalMinutes, MinUpdateCheckIntervalMinutes, MaxUpdateCheckIntervalMinutes, DefaultUpdateCheckIntervalMinutes},
		{&s.StoreRetentionDays, MinStoreRetentionDays, MaxStoreRetentionDays, DefaultStoreRetentionDays},
		{&s.TrashRetentionDays, MinTrashRetentionDays, MaxTrashRetentionDays, DefaultTrashRetentionDays},
		{&s.HistoryEventsKept, MinHistoryEventsKept, MaxHistoryEventsKept, DefaultHistoryEventsKept},
	}
}

func normBools(s *Settings) []normBool {
	return []normBool{
		{&s.NotifyModUpdates, on},
		{&s.ConfirmRemovals, on},
		{&s.BackgroundBadgeChecks, on},
		{&s.NotifyDownloadFinished, off},
		{&s.NotifyDownloadFailed, on},
		{&s.NotifyRunCrashed, on},
		{&s.ShowAuthorOnCards, on},
		{&s.MarkLocalMods, on},
		{&s.ReuseFomodChoices, on},
		{&s.DriftChecks, on},
		{&s.AutoInstallMortarUpdates, on},
		{&s.ShareIncludeDisabledMods, on},
		{&s.ShareIncludeFomodChoices, on},
		{&s.ShareIncludeNotes, on},
		{&s.ShareIncludeConfigFiles, on},
		{&s.ShareIncludeLibraries, on},
		{&s.ShareIncludeProblemChoices, on},
		{&s.ShowSmapiConsole, on},
	}
}

func validatePrefs(s Settings) error {
	for _, check := range prefChecks(s) {
		if err := check(); err != nil {
			return err
		}
	}
	for _, g := range s.Games {
		if g == nil {
			continue
		}
		if err := validateGame(*g); err != nil {
			return err
		}
	}
	return nil
}

type prefCheck func() error

func enumCheck(value string, values []string, msg string) prefCheck {
	return func() error {
		if slices.Contains(values, value) {
			return nil
		}
		return fmt.Errorf("%s, got %q", msg, value)
	}
}

func rangeCheck(value, lo, hi int, name, unit string) prefCheck {
	return func() error {
		if value >= lo && value <= hi {
			return nil
		}
		return fmt.Errorf("%s must be %d to %d%s, got %d", name, lo, hi, unit, value)
	}
}

// prefChecks lists the checks in the order the first invalid field is reported.
func prefChecks(s Settings) []prefCheck {
	return []prefCheck{
		enumCheck(s.OnPlay, onPlayValues, "on play must be stay, minimise or hide"),
		rangeCheck(s.ParallelDownloads, MinParallelDownloads, MaxParallelDownloads, "parallel downloads", ""),
		rangeCheck(s.DownloadSpeedLimit, MinDownloadSpeedLimit, MaxDownloadSpeedLimit, "download speed limit", " MB/s"),
		rangeCheck(s.UpdateCheckIntervalMinutes, MinUpdateCheckIntervalMinutes, MaxUpdateCheckIntervalMinutes, "update check interval", " minutes"),
		rangeCheck(s.StoreRetentionDays, MinStoreRetentionDays, MaxStoreRetentionDays, "store retention days", ""),
		enumCheck(s.DefaultModsView, modsViewValues, "default mods view must be grid or list"),
		enumCheck(s.ImplicitModsView, implicitModsValues, "implicit mods view must be shown, nested or hidden"),
		enumCheck(s.StartScreen, startScreenValues, "start screen must be last or gameselect"),
		enumCheck(s.Antivirus, antivirusValues, "antivirus must be automatic, clamd, command or off"),
		enumCheck(s.Dates, datesValues, "dates must be relative or absolute"),
		enumCheck(s.Graphics, graphicsValues, "graphics must be automatic, smooth or compatible"),
		rangeCheck(s.TrashRetentionDays, MinTrashRetentionDays, MaxTrashRetentionDays, "trash retention days", ""),
		rangeCheck(s.HistoryEventsKept, MinHistoryEventsKept, MaxHistoryEventsKept, "history events kept", ""),
		enumCheck(s.Density, densityValues, "density must be comfortable or compact"),
		enumCheck(s.Theme, themeValues, "theme must be dark, light or system"),
		enumCheck(s.GridCardSize, gridCardValues, "grid card size must be small, medium or large"),
		enumCheck(s.ReduceMotion, reduceMotionValues, "reduce motion must be system, always or never"),
		enumCheck(s.ProfileHero, heroValues, "profile hero must be full, compact or hidden"),
		enumCheck(s.ProfileOrder, profileOrderValues, "profile order must be manual, name or lastPlayed"),
		enumCheck(s.ThunderstoreHost, thunderstoreHostValues, "thunderstore host must be auto or one of its download hosts"),
		enumCheck(s.AutoRetryDownloads, autoRetryValues, "auto-retry downloads must be off, 1 or 3"),
		enumCheck(s.SidebarBadges, sidebarBadgesValues, "sidebar badges must be problemsAndUpdates, problems or off"),
		enumCheck(s.ExtensionConnection, extensionConnectionValues, "extension connection must be allow or off"),
		enumCheck(s.SmapiBuilds, smapiBuildsValues, "smapi builds must be never, show or include"),
	}
}

var (
	onPlayValues              = []string{OnPlayStay, OnPlayMinimise, OnPlayHide}
	backupBeforePlayValues    = []string{BackupBeforePlayChanged, BackupBeforePlayAlways, BackupBeforePlayNever}
	cosmeticValues            = []string{CosmeticCollapsed, CosmeticExpanded, CosmeticHidden}
	startScreenValues         = []string{StartScreenLast, StartScreenGameSelect}
	antivirusValues           = []string{AntivirusAutomatic, AntivirusClamd, AntivirusCommand, AntivirusOff}
	datesValues               = []string{DatesRelative, DatesAbsolute}
	graphicsValues            = []string{GraphicsAutomatic, GraphicsSmooth, GraphicsCompatible}
	modsViewValues            = []string{ModsViewGrid, ModsViewList}
	implicitModsValues        = []string{ImplicitModsShown, ImplicitModsNested, ImplicitModsHidden}
	densityValues             = []string{DensityComfortable, DensityCompact}
	themeValues               = []string{ThemeDark, ThemeLight, ThemeSystem}
	gridCardValues            = []string{GridCardSmall, GridCardMedium, GridCardLarge}
	reduceMotionValues        = []string{ReduceMotionSystem, ReduceMotionAlways, ReduceMotionNever}
	heroValues                = []string{HeroFull, HeroCompact, HeroHidden}
	enableReqValues           = []string{EnableReqAlways, EnableReqAsk, EnableReqNever}
	missingReqValues          = []string{MissingReqAsk, MissingReqAutodownload, MissingReqNever}
	smapiBuildsValues         = []string{SmapiBuildsNever, SmapiBuildsShow, SmapiBuildsInclude}
	launchMethodValues        = []string{LaunchSteam, LaunchDirect}
	startFirstValues          = []string{StartFirstOff, StartFirstSteamVR, StartFirstALVR}
	consoleLevelValues        = []string{ConsoleLevelTrace, ConsoleLevelDebug, ConsoleLevelInfo, ConsoleLevelWarn, ConsoleLevelError}
	profileOrderValues        = []string{ProfileOrderManual, ProfileOrderName, ProfileOrderLastPlayed}
	thunderstoreHostValues    = []string{ThunderstoreHostAuto, ThunderstoreHostC, ThunderstoreHostG, ThunderstoreHostH}
	autoRetryValues           = []string{AutoRetryOff, AutoRetry1, AutoRetry3}
	sidebarBadgesValues       = []string{SidebarBadgesAll, SidebarBadgesProblems, SidebarBadgesOff}
	conflictScanValues        = []string{ConflictScanFull, ConflictScanSkipImages}
	extensionConnectionValues = []string{ExtensionAllow, ExtensionOff}
	oldFilesValues            = []string{OldFilesAsk, OldFilesDelete, OldFilesKeep}
	gameSettingsModeValues    = []string{GameSettingsEdit, GameSettingsWarn}
	cacheClearingValues       = []string{CacheClearingAuto, CacheClearingOff}
	syncModeValues            = []string{SyncFollow, SyncOwn, SyncOff}
	dependencyVersionValues   = []string{DependencyNewest, DependencyExact}
)

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "true", "on", "1", "yes":
		return true, nil
	case "false", "off", "0", "no":
		return false, nil
	}
	return false, fmt.Errorf("must be true or false, got %q", raw)
}

// ToggleOn is true when a *bool setting is on or omitted.
func ToggleOn(v *bool) bool { return v == nil || *v }

// StoreUnusedFor is how long an unused store item is kept; 0 days means forever.
func (s Settings) StoreUnusedFor() time.Duration {
	if s.StoreRetentionDays <= 0 {
		return 0
	}
	return time.Duration(s.StoreRetentionDays) * 24 * time.Hour
}

// TrashKeepFor is how long a deleted profile stays restorable.
func (s Settings) TrashKeepFor() time.Duration {
	n := s.TrashRetentionDays
	if n < MinTrashRetentionDays || n > MaxTrashRetentionDays {
		n = DefaultTrashRetentionDays
	}
	return time.Duration(n) * 24 * time.Hour
}

// ShouldBackupBeforePlay reports whether a launch backup should run for this mode.
func ShouldBackupBeforePlay(mode string, modsChanged, gameVersionChanged bool) bool {
	switch mode {
	case BackupBeforePlayAlways:
		return true
	case BackupBeforePlayNever:
		return false
	default:
		return modsChanged || gameVersionChanged
	}
}

// AutoEnableRequirements is whether enabling a mod also turns on its required dependencies already in the profile.
func (g GameSettings) AutoEnableRequirements() bool {
	return g.EnableRequirements != EnableReqNever && g.EnableRequirements != EnableReqAsk
}

// DriftChecksOn is whether Mortar looks for mods changed outside Mortar.
func (s Settings) DriftChecksOn() bool {
	return ToggleOn(s.DriftChecks)
}

// ReuseFomod is whether saved FOMOD plugin choices skip the wizard when they still match.
func (s Settings) ReuseFomod() bool {
	return ToggleOn(s.ReuseFomodChoices)
}

// AutoInstallMortar is whether a found Mortar update is staged without asking.
func (s Settings) AutoInstallMortar() bool {
	return ToggleOn(s.AutoInstallMortarUpdates)
}

// WatchedFolders lists the extra archive folders: absolute paths only, each once.
func (s Settings) WatchedFolders() []string {
	return pathList(s.WatchFolders)
}

// moveAutoInstallFolders makes each folder the player set to auto-install an extra mods folder of the game they last
// had open, the one it installed into, and drops it from the archive folders it could also be in.
func moveAutoInstallFolders(s *Settings) {
	auto := pathList(s.AutoInstallFolders)
	// With no game open yet the folders stay until one is.
	if len(auto) == 0 || s.LastGame == "" {
		if len(auto) == 0 {
			s.AutoInstallFolders = ""
		}
		return
	}
	s.AutoInstallFolders = ""
	gp := s.GamePrefs(s.LastGame)
	extra := gp.ExtraModsFolderList()
	for _, d := range auto {
		if !slices.Contains(extra, d) {
			extra = append(extra, d)
		}
	}
	gp.ExtraModsFolder = strings.Join(extra, string(filepath.ListSeparator))
	putGame(s, s.LastGame, gp)
	watched := slices.DeleteFunc(pathList(s.WatchFolders), func(d string) bool { return slices.Contains(auto, d) })
	s.WatchFolders = strings.Join(watched, string(filepath.ListSeparator))
}

// SyncFolderFor is the folder the game's profiles sync into; empty means sync is off for the game.
func (s Settings) SyncFolderFor(game string) string {
	g := s.GamePrefs(game)
	switch g.SyncMode {
	case SyncOwn:
		return g.SyncOwnFolder
	case SyncOff:
		return ""
	}
	return s.SyncFolder
}

// ExtraModsFolderList lists the game's extra mods folders: absolute paths only, each once.
func (g GameSettings) ExtraModsFolderList() []string {
	return pathList(g.ExtraModsFolder)
}

// SourceOrderIDs lists the game's preferred source ids, first first.
func (g GameSettings) SourceOrderIDs() []string {
	return commaList(g.SourceOrder)
}

// SourcesOffIDs lists the source ids the player turned off for the game.
func (g GameSettings) SourcesOffIDs() []string {
	return commaList(g.SourcesOff)
}

func commaList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' })
}

// ExtraModsProfile is the id of the profile the extra mods folder dir installs into; empty is the open profile.
func (g GameSettings) ExtraModsProfile(dir string) string {
	return g.ExtraModsProfiles[dir]
}

func pathList(joined string) []string {
	var out []string
	for _, d := range filepath.SplitList(joined) {
		if d = strings.TrimSpace(d); filepath.IsAbs(d) && !slices.Contains(out, d) {
			out = append(out, d)
		}
	}
	return out
}

// ArchiveDir is where download zips land; empty means the data folder's downloads directory.
func (s Settings) ArchiveDir() string {
	return strings.TrimSpace(s.DownloadFolder)
}
