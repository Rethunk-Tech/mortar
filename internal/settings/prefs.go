package settings

import (
	"fmt"
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

	StartScreenLast       = "last"
	StartScreenGameSelect = "gameselect"

	DatesRelative = "relative"
	DatesAbsolute = "absolute"

	ModsViewGrid = "grid"
	ModsViewList = "list"

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

	ConsoleLevelTrace = "trace"
	ConsoleLevelDebug = "debug"
	ConsoleLevelInfo  = "info"
	ConsoleLevelWarn  = "warn"
	ConsoleLevelError = "error"

	ProfileOrderManual     = "manual"
	ProfileOrderName       = "name"
	ProfileOrderLastPlayed = "lastPlayed"

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

	DefaultRunsKept                   = 20
	DefaultConsoleLogCap              = 20000
	DefaultParallelDownloads          = 3
	DefaultUpdateCheckIntervalMinutes = 60
	DefaultStoreRetentionDays         = 30
	DefaultTrashRetentionDays         = 30
	DefaultHistoryEventsKept          = 200
	DefaultLaunchBackupsKept          = 5

	MinRunsKept                   = 1
	MaxRunsKept                   = 100
	MinConsoleLogCap              = 1000
	MaxConsoleLogCap              = 100000
	MinParallelDownloads          = 1
	MaxParallelDownloads          = 8
	MinUpdateCheckIntervalMinutes = 15
	MaxUpdateCheckIntervalMinutes = 24 * 60
	MinStoreRetentionDays         = 0
	MaxStoreRetentionDays         = 3650
	MinTrashRetentionDays         = 1
	MaxTrashRetentionDays         = 365
	MinHistoryEventsKept          = 20
	MaxHistoryEventsKept          = 2000
	MinLaunchBackupsKept          = 1
	MaxLaunchBackupsKept          = 50
)

func off() *bool { v := false; return &v }

func defaultPrefs() Settings {
	return Settings{
		OnPlay:                     OnPlayStay,
		ParallelDownloads:          DefaultParallelDownloads,
		UpdateCheckIntervalMinutes: DefaultUpdateCheckIntervalMinutes,
		NotifyModUpdates:           off(),
		UpdateDigest:               UpdateDigestDaily,
		KeepDownloadArchives:       false,
		StoreRetentionDays:         DefaultStoreRetentionDays,
		DefaultModsView:            ModsViewGrid,
		ConfirmRemovals:            on(),
		BackgroundBadgeChecks:      on(),
		StartScreen:                StartScreenLast,
		Dates:                      DatesRelative,
		TrashRetentionDays:         DefaultTrashRetentionDays,
		HistoryEventsKept:          DefaultHistoryEventsKept,
		NotifyDownloadFinished:     on(),
		NotifyDownloadFailed:       on(),
		NotifyRunCrashed:           on(),
		Density:                    DensityComfortable,
		Theme:                      ThemeDark,
		GridCardSize:               GridCardMedium,
		ShowAuthorOnCards:          on(),
		ReduceMotion:               ReduceMotionSystem,
		ProfileHero:                HeroFull,
		ReuseFomodChoices:          on(),
		DriftChecks:                on(),
		AutoInstallMortarUpdates:   on(),
		AutoTrackNexus:             false,
		LanName:                    "",
		LanAutoAcceptSameAccount:   false,
		DownloadFolder:             "",
		ProfileOrder:               ProfileOrderManual,
		AutoRetryDownloads:         AutoRetryOff,
		PauseDownloadsWhilePlaying: false,
		SidebarBadges:              SidebarBadgesAll,
		ShareIncludeDisabledMods:   off(),
		ShareIncludeFomodChoices:   on(),
		ShareIncludeNotes:          on(),
		ShareIncludeConfigFiles:    on(),
		VerifyNexusMD5:             false,
		LaunchAtLogin:              false,
		StartMinimised:             false,
		RememberWindow:             false,
		ExtensionConnection:        ExtensionAllow,
		Games:                      map[string]*GameSettings{},
	}
}

func normalizePrefs(s *Settings) {
	if !slices.Contains(onPlayValues, s.OnPlay) {
		s.OnPlay = OnPlayStay
	}
	if s.ParallelDownloads < MinParallelDownloads || s.ParallelDownloads > MaxParallelDownloads {
		s.ParallelDownloads = DefaultParallelDownloads
	}
	if s.UpdateCheckIntervalMinutes < MinUpdateCheckIntervalMinutes || s.UpdateCheckIntervalMinutes > MaxUpdateCheckIntervalMinutes {
		s.UpdateCheckIntervalMinutes = DefaultUpdateCheckIntervalMinutes
	}
	if s.NotifyModUpdates == nil {
		s.NotifyModUpdates = off()
	}
	normalizeUpdateDigest(s)
	if s.StoreRetentionDays < MinStoreRetentionDays || s.StoreRetentionDays > MaxStoreRetentionDays {
		s.StoreRetentionDays = DefaultStoreRetentionDays
	}
	if !slices.Contains(modsViewValues, s.DefaultModsView) {
		s.DefaultModsView = ModsViewGrid
	}
	if s.ConfirmRemovals == nil {
		s.ConfirmRemovals = on()
	}
	if s.BackgroundBadgeChecks == nil {
		s.BackgroundBadgeChecks = on()
	}
	if !slices.Contains(startScreenValues, s.StartScreen) {
		s.StartScreen = StartScreenLast
	}
	if !slices.Contains(datesValues, s.Dates) {
		s.Dates = DatesRelative
	}
	if s.TrashRetentionDays < MinTrashRetentionDays || s.TrashRetentionDays > MaxTrashRetentionDays {
		s.TrashRetentionDays = DefaultTrashRetentionDays
	}
	if s.HistoryEventsKept < MinHistoryEventsKept || s.HistoryEventsKept > MaxHistoryEventsKept {
		s.HistoryEventsKept = DefaultHistoryEventsKept
	}
	if s.NotifyDownloadFinished == nil {
		s.NotifyDownloadFinished = on()
	}
	if s.NotifyDownloadFailed == nil {
		s.NotifyDownloadFailed = on()
	}
	if s.NotifyRunCrashed == nil {
		s.NotifyRunCrashed = on()
	}
	if !slices.Contains(densityValues, s.Density) {
		s.Density = DensityComfortable
	}
	if !slices.Contains(themeValues, s.Theme) {
		s.Theme = ThemeDark
	}
	if !slices.Contains(gridCardValues, s.GridCardSize) {
		s.GridCardSize = GridCardMedium
	}
	if s.ShowAuthorOnCards == nil {
		s.ShowAuthorOnCards = on()
	}
	if !slices.Contains(reduceMotionValues, s.ReduceMotion) {
		s.ReduceMotion = ReduceMotionSystem
	}
	if !slices.Contains(heroValues, s.ProfileHero) {
		s.ProfileHero = HeroFull
	}
	if s.ReuseFomodChoices == nil {
		s.ReuseFomodChoices = on()
	}
	if s.DriftChecks == nil {
		s.DriftChecks = on()
	}
	if s.AutoInstallMortarUpdates == nil {
		s.AutoInstallMortarUpdates = on()
	}
	if !slices.Contains(profileOrderValues, s.ProfileOrder) {
		s.ProfileOrder = ProfileOrderManual
	}
	if !slices.Contains(autoRetryValues, s.AutoRetryDownloads) {
		s.AutoRetryDownloads = AutoRetryOff
	}
	if !slices.Contains(sidebarBadgesValues, s.SidebarBadges) {
		s.SidebarBadges = SidebarBadgesAll
	}
	if s.ShareIncludeDisabledMods == nil {
		s.ShareIncludeDisabledMods = off()
	}
	if s.ShareIncludeFomodChoices == nil {
		s.ShareIncludeFomodChoices = on()
	}
	if s.ShareIncludeNotes == nil {
		s.ShareIncludeNotes = on()
	}
	if s.ShareIncludeConfigFiles == nil {
		s.ShareIncludeConfigFiles = on()
	}
	if !slices.Contains(extensionConnectionValues, s.ExtensionConnection) {
		s.ExtensionConnection = ExtensionAllow
	}
	if s.Games == nil {
		s.Games = map[string]*GameSettings{}
	}
	for id, g := range s.Games {
		if g == nil {
			d := defaultGameSettings()
			s.Games[id] = &d
			continue
		}
		normalizeGame(g)
	}
}

func validatePrefs(s Settings) error {
	if !slices.Contains(onPlayValues, s.OnPlay) {
		return fmt.Errorf("on play must be stay, minimise or hide, got %q", s.OnPlay)
	}
	if s.ParallelDownloads < MinParallelDownloads || s.ParallelDownloads > MaxParallelDownloads {
		return fmt.Errorf("parallel downloads must be %d to %d, got %d", MinParallelDownloads, MaxParallelDownloads, s.ParallelDownloads)
	}
	if s.UpdateCheckIntervalMinutes < MinUpdateCheckIntervalMinutes || s.UpdateCheckIntervalMinutes > MaxUpdateCheckIntervalMinutes {
		return fmt.Errorf("update check interval must be %d to %d minutes, got %d", MinUpdateCheckIntervalMinutes, MaxUpdateCheckIntervalMinutes, s.UpdateCheckIntervalMinutes)
	}
	if s.StoreRetentionDays < MinStoreRetentionDays || s.StoreRetentionDays > MaxStoreRetentionDays {
		return fmt.Errorf("store retention days must be %d to %d, got %d", MinStoreRetentionDays, MaxStoreRetentionDays, s.StoreRetentionDays)
	}
	if !slices.Contains(modsViewValues, s.DefaultModsView) {
		return fmt.Errorf("default mods view must be grid or list, got %q", s.DefaultModsView)
	}
	if !slices.Contains(startScreenValues, s.StartScreen) {
		return fmt.Errorf("start screen must be last or gameselect, got %q", s.StartScreen)
	}
	if !slices.Contains(datesValues, s.Dates) {
		return fmt.Errorf("dates must be relative or absolute, got %q", s.Dates)
	}
	if s.TrashRetentionDays < MinTrashRetentionDays || s.TrashRetentionDays > MaxTrashRetentionDays {
		return fmt.Errorf("trash retention days must be %d to %d, got %d", MinTrashRetentionDays, MaxTrashRetentionDays, s.TrashRetentionDays)
	}
	if s.HistoryEventsKept < MinHistoryEventsKept || s.HistoryEventsKept > MaxHistoryEventsKept {
		return fmt.Errorf("history events kept must be %d to %d, got %d", MinHistoryEventsKept, MaxHistoryEventsKept, s.HistoryEventsKept)
	}
	if !slices.Contains(densityValues, s.Density) {
		return fmt.Errorf("density must be comfortable or compact, got %q", s.Density)
	}
	if !slices.Contains(themeValues, s.Theme) {
		return fmt.Errorf("theme must be dark, light or system, got %q", s.Theme)
	}
	if !slices.Contains(gridCardValues, s.GridCardSize) {
		return fmt.Errorf("grid card size must be small, medium or large, got %q", s.GridCardSize)
	}
	if !slices.Contains(reduceMotionValues, s.ReduceMotion) {
		return fmt.Errorf("reduce motion must be system, always or never, got %q", s.ReduceMotion)
	}
	if !slices.Contains(heroValues, s.ProfileHero) {
		return fmt.Errorf("profile hero must be full, compact or hidden, got %q", s.ProfileHero)
	}
	if !slices.Contains(profileOrderValues, s.ProfileOrder) {
		return fmt.Errorf("profile order must be manual, name or lastPlayed, got %q", s.ProfileOrder)
	}
	if !slices.Contains(autoRetryValues, s.AutoRetryDownloads) {
		return fmt.Errorf("auto-retry downloads must be off, 1 or 3, got %q", s.AutoRetryDownloads)
	}
	if !slices.Contains(sidebarBadgesValues, s.SidebarBadges) {
		return fmt.Errorf("sidebar badges must be problemsAndUpdates, problems or off, got %q", s.SidebarBadges)
	}
	if !slices.Contains(extensionConnectionValues, s.ExtensionConnection) {
		return fmt.Errorf("extension connection must be allow or off, got %q", s.ExtensionConnection)
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

var (
	onPlayValues              = []string{OnPlayStay, OnPlayMinimise, OnPlayHide}
	backupBeforePlayValues    = []string{BackupBeforePlayChanged, BackupBeforePlayAlways, BackupBeforePlayNever}
	cosmeticValues            = []string{CosmeticCollapsed, CosmeticExpanded, CosmeticHidden}
	startScreenValues         = []string{StartScreenLast, StartScreenGameSelect}
	datesValues               = []string{DatesRelative, DatesAbsolute}
	modsViewValues            = []string{ModsViewGrid, ModsViewList}
	densityValues             = []string{DensityComfortable, DensityCompact}
	themeValues               = []string{ThemeDark, ThemeLight, ThemeSystem}
	gridCardValues            = []string{GridCardSmall, GridCardMedium, GridCardLarge}
	reduceMotionValues        = []string{ReduceMotionSystem, ReduceMotionAlways, ReduceMotionNever}
	heroValues                = []string{HeroFull, HeroCompact, HeroHidden}
	enableReqValues           = []string{EnableReqAlways, EnableReqAsk, EnableReqNever}
	missingReqValues          = []string{MissingReqAsk, MissingReqAutodownload, MissingReqNever}
	smapiBuildsValues         = []string{SmapiBuildsNever, SmapiBuildsShow, SmapiBuildsInclude}
	launchMethodValues        = []string{LaunchSteam, LaunchDirect}
	consoleLevelValues        = []string{ConsoleLevelTrace, ConsoleLevelDebug, ConsoleLevelInfo, ConsoleLevelWarn, ConsoleLevelError}
	profileOrderValues        = []string{ProfileOrderManual, ProfileOrderName, ProfileOrderLastPlayed}
	autoRetryValues           = []string{AutoRetryOff, AutoRetry1, AutoRetry3}
	sidebarBadgesValues       = []string{SidebarBadgesAll, SidebarBadgesProblems, SidebarBadgesOff}
	conflictScanValues        = []string{ConflictScanFull, ConflictScanSkipImages}
	extensionConnectionValues = []string{ExtensionAllow, ExtensionOff}
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

// PrefKey is one CLI/settings get|set name.
type PrefKey struct {
	Key   string
	Scope string
	Get   func(Settings, string) string
	Apply func(*Settings, string, string) error
}

// PrefKeys are the user-facing settings the CLI can get and set.
func PrefKeys() []PrefKey {
	out := make([]PrefKey, 0, len(registry))
	for _, p := range registry {
		out = append(out, PrefKey{
			Key:   p.spec.Key,
			Scope: p.spec.Scope,
			Get:   p.get,
			Apply: p.set,
		})
	}
	return out
}

// ToggleOn is true when a *bool setting is on or omitted.
func ToggleOn(v *bool) bool { return v == nil || *v }

// UpdateCheckEvery is the background mod-update interval.
func (s Settings) UpdateCheckEvery() time.Duration {
	n := s.UpdateCheckIntervalMinutes
	if n < MinUpdateCheckIntervalMinutes || n > MaxUpdateCheckIntervalMinutes {
		n = DefaultUpdateCheckIntervalMinutes
	}
	return time.Duration(n) * time.Minute
}

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

// ShowConsoleWindow is whether a direct SMAPI launch keeps a console window.
func (g GameSettings) ShowConsoleWindow() bool {
	return ToggleOn(g.ShowSmapiConsole)
}

// ArchiveDir is where download zips land; empty means the data folder's downloads directory.
func (s Settings) ArchiveDir() string {
	return strings.TrimSpace(s.DownloadFolder)
}
