package settings

import (
	"fmt"
	"slices"
	"strconv"
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
		OnPlay:                      OnPlayStay,
		BackupBeforePlay:            BackupBeforePlayChanged,
		LaunchBackupsKept:           DefaultLaunchBackupsKept,
		UpdateModsBeforePlayDefault: false,
		RunsKept:                    DefaultRunsKept,
		ConsoleLogCap:               DefaultConsoleLogCap,
		ParallelDownloads:           DefaultParallelDownloads,
		UpdateCheckIntervalMinutes:  DefaultUpdateCheckIntervalMinutes,
		NotifyModUpdates:            off(),
		KeepDownloadArchives:        false,
		StoreRetentionDays:          DefaultStoreRetentionDays,
		NxmDefaultProfile:           "",
		DefaultModsView:             ModsViewGrid,
		ConfirmRemovals:             on(),
		CosmeticConflicts:           CosmeticCollapsed,
		BackgroundBadgeChecks:       on(),
		StartScreen:                 StartScreenLast,
		Dates:                       DatesRelative,
		TrashRetentionDays:          DefaultTrashRetentionDays,
		HistoryEventsKept:           DefaultHistoryEventsKept,
		NotifyDownloadFinished:      on(),
		NotifyDownloadFailed:        on(),
		NotifyRunCrashed:            on(),
		Density:                     DensityComfortable,
		GridCardSize:                GridCardMedium,
		ShowAuthorOnCards:           on(),
		ReduceMotion:                ReduceMotionSystem,
		ProfileHero:                 HeroFull,
		EnableRequirements:          EnableReqAlways,
		MissingRequirements:         MissingReqAsk,
		ReuseFomodChoices:           on(),
		DriftChecks:                 on(),
		SmapiBuilds:                 SmapiBuildsShow,
		AutoInstallMortarUpdates:    on(),
		AutoTrackNexus:              false,
		DefaultLaunchMethod:         LaunchSteam,
		ShowSmapiConsole:            on(),
		ConsoleLevel:                ConsoleLevelInfo,
		ConsoleTimestamps:           on(),
		ConsoleFollow:               on(),
		LanName:                     "",
		LanAutoAcceptSameAccount:    false,
		DownloadFolder:              "",
	}
}

func normalizePrefs(s *Settings) {
	if !slices.Contains(onPlayValues, s.OnPlay) {
		s.OnPlay = OnPlayStay
	}
	if !slices.Contains(backupBeforePlayValues, s.BackupBeforePlay) {
		s.BackupBeforePlay = BackupBeforePlayChanged
	}
	if s.LaunchBackupsKept < MinLaunchBackupsKept || s.LaunchBackupsKept > MaxLaunchBackupsKept {
		s.LaunchBackupsKept = DefaultLaunchBackupsKept
	}
	if s.RunsKept < MinRunsKept || s.RunsKept > MaxRunsKept {
		s.RunsKept = DefaultRunsKept
	}
	if s.ConsoleLogCap < MinConsoleLogCap || s.ConsoleLogCap > MaxConsoleLogCap {
		s.ConsoleLogCap = DefaultConsoleLogCap
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
	if s.StoreRetentionDays < MinStoreRetentionDays || s.StoreRetentionDays > MaxStoreRetentionDays {
		s.StoreRetentionDays = DefaultStoreRetentionDays
	}
	if !slices.Contains(modsViewValues, s.DefaultModsView) {
		s.DefaultModsView = ModsViewGrid
	}
	if s.ConfirmRemovals == nil {
		s.ConfirmRemovals = on()
	}
	if !slices.Contains(cosmeticValues, s.CosmeticConflicts) {
		s.CosmeticConflicts = CosmeticCollapsed
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
	if !slices.Contains(enableReqValues, s.EnableRequirements) {
		s.EnableRequirements = EnableReqAlways
	}
	if !slices.Contains(missingReqValues, s.MissingRequirements) {
		s.MissingRequirements = MissingReqAsk
	}
	if s.ReuseFomodChoices == nil {
		s.ReuseFomodChoices = on()
	}
	if s.DriftChecks == nil {
		s.DriftChecks = on()
	}
	if !slices.Contains(smapiBuildsValues, s.SmapiBuilds) {
		s.SmapiBuilds = SmapiBuildsShow
	}
	if s.AutoInstallMortarUpdates == nil {
		s.AutoInstallMortarUpdates = on()
	}
	if !slices.Contains(launchMethodValues, s.DefaultLaunchMethod) {
		s.DefaultLaunchMethod = LaunchSteam
	}
	if s.ShowSmapiConsole == nil {
		s.ShowSmapiConsole = on()
	}
	if !slices.Contains(consoleLevelValues, s.ConsoleLevel) {
		s.ConsoleLevel = ConsoleLevelInfo
	}
	if s.ConsoleTimestamps == nil {
		s.ConsoleTimestamps = on()
	}
	if s.ConsoleFollow == nil {
		s.ConsoleFollow = on()
	}
}

func validatePrefs(s Settings) error {
	if !slices.Contains(onPlayValues, s.OnPlay) {
		return fmt.Errorf("on play must be stay, minimise or hide, got %q", s.OnPlay)
	}
	if !slices.Contains(backupBeforePlayValues, s.BackupBeforePlay) {
		return fmt.Errorf("backup before play must be changed, always or never, got %q", s.BackupBeforePlay)
	}
	if s.LaunchBackupsKept < MinLaunchBackupsKept || s.LaunchBackupsKept > MaxLaunchBackupsKept {
		return fmt.Errorf("launch backups kept must be %d to %d, got %d", MinLaunchBackupsKept, MaxLaunchBackupsKept, s.LaunchBackupsKept)
	}
	if s.RunsKept < MinRunsKept || s.RunsKept > MaxRunsKept {
		return fmt.Errorf("runs kept must be %d to %d, got %d", MinRunsKept, MaxRunsKept, s.RunsKept)
	}
	if s.ConsoleLogCap < MinConsoleLogCap || s.ConsoleLogCap > MaxConsoleLogCap {
		return fmt.Errorf("console log cap must be %d to %d, got %d", MinConsoleLogCap, MaxConsoleLogCap, s.ConsoleLogCap)
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
	if !slices.Contains(cosmeticValues, s.CosmeticConflicts) {
		return fmt.Errorf("cosmetic conflicts must be collapsed, expanded or hidden, got %q", s.CosmeticConflicts)
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
	if !slices.Contains(gridCardValues, s.GridCardSize) {
		return fmt.Errorf("grid card size must be small, medium or large, got %q", s.GridCardSize)
	}
	if !slices.Contains(reduceMotionValues, s.ReduceMotion) {
		return fmt.Errorf("reduce motion must be system, always or never, got %q", s.ReduceMotion)
	}
	if !slices.Contains(heroValues, s.ProfileHero) {
		return fmt.Errorf("profile hero must be full, compact or hidden, got %q", s.ProfileHero)
	}
	if !slices.Contains(enableReqValues, s.EnableRequirements) {
		return fmt.Errorf("enable requirements must be always, ask or never, got %q", s.EnableRequirements)
	}
	if !slices.Contains(missingReqValues, s.MissingRequirements) {
		return fmt.Errorf("missing requirements must be ask, autodownload or never, got %q", s.MissingRequirements)
	}
	if !slices.Contains(smapiBuildsValues, s.SmapiBuilds) {
		return fmt.Errorf("smapi builds must be never, show or include, got %q", s.SmapiBuilds)
	}
	if !slices.Contains(launchMethodValues, s.DefaultLaunchMethod) {
		return fmt.Errorf("default launch method must be steam or direct, got %q", s.DefaultLaunchMethod)
	}
	if !slices.Contains(consoleLevelValues, s.ConsoleLevel) {
		return fmt.Errorf("console level must be trace, debug, info, warn or error, got %q", s.ConsoleLevel)
	}
	return nil
}

var (
	onPlayValues           = []string{OnPlayStay, OnPlayMinimise, OnPlayHide}
	backupBeforePlayValues = []string{BackupBeforePlayChanged, BackupBeforePlayAlways, BackupBeforePlayNever}
	cosmeticValues         = []string{CosmeticCollapsed, CosmeticExpanded, CosmeticHidden}
	startScreenValues      = []string{StartScreenLast, StartScreenGameSelect}
	datesValues            = []string{DatesRelative, DatesAbsolute}
	modsViewValues         = []string{ModsViewGrid, ModsViewList}
	densityValues          = []string{DensityComfortable, DensityCompact}
	gridCardValues         = []string{GridCardSmall, GridCardMedium, GridCardLarge}
	reduceMotionValues     = []string{ReduceMotionSystem, ReduceMotionAlways, ReduceMotionNever}
	heroValues             = []string{HeroFull, HeroCompact, HeroHidden}
	enableReqValues        = []string{EnableReqAlways, EnableReqAsk, EnableReqNever}
	missingReqValues       = []string{MissingReqAsk, MissingReqAutodownload, MissingReqNever}
	smapiBuildsValues      = []string{SmapiBuildsNever, SmapiBuildsShow, SmapiBuildsInclude}
	launchMethodValues     = []string{LaunchSteam, LaunchDirect}
	consoleLevelValues     = []string{ConsoleLevelTrace, ConsoleLevelDebug, ConsoleLevelInfo, ConsoleLevelWarn, ConsoleLevelError}
)

// PrefKey is one CLI/settings get|set name.
type PrefKey struct {
	Key   string
	Get   func(Settings) string
	Apply func(*Settings, string) error
}

// PrefKeys are the user-facing settings the CLI can get and set.
func PrefKeys() []PrefKey {
	return []PrefKey{
		{Key: "onPlay", Get: func(s Settings) string { return s.OnPlay }, Apply: enumApply(onPlayValues, func(s *Settings, v string) { s.OnPlay = v })},
		{Key: "backupBeforePlay", Get: func(s Settings) string { return s.BackupBeforePlay }, Apply: enumApply(backupBeforePlayValues, func(s *Settings, v string) { s.BackupBeforePlay = v })},
		{Key: "launchBackupsKept", Get: intGet(func(s Settings) int { return s.LaunchBackupsKept }), Apply: intApply(func(s *Settings, n int) { s.LaunchBackupsKept = n })},
		{Key: "updateModsBeforePlayDefault", Get: boolGet(func(s Settings) bool { return s.UpdateModsBeforePlayDefault }), Apply: boolApply(func(s *Settings, on bool) { s.UpdateModsBeforePlayDefault = on })},
		{Key: "runsKept", Get: intGet(func(s Settings) int { return s.RunsKept }), Apply: intApply(func(s *Settings, n int) { s.RunsKept = n })},
		{Key: "consoleLogCap", Get: intGet(func(s Settings) int { return s.ConsoleLogCap }), Apply: intApply(func(s *Settings, n int) { s.ConsoleLogCap = n })},
		{Key: "parallelDownloads", Get: intGet(func(s Settings) int { return s.ParallelDownloads }), Apply: intApply(func(s *Settings, n int) { s.ParallelDownloads = n })},
		{Key: "updateCheckIntervalMinutes", Get: intGet(func(s Settings) int { return s.UpdateCheckIntervalMinutes }), Apply: intApply(func(s *Settings, n int) { s.UpdateCheckIntervalMinutes = n })},
		{Key: "checkModUpdatesOnStart", Get: ptrGet(func(s Settings) *bool { return s.CheckModUpdatesOnStart }), Apply: ptrApply(func(s *Settings, on bool) { s.CheckModUpdatesOnStart = &on })},
		{Key: "notifyModUpdates", Get: ptrGet(func(s Settings) *bool { return s.NotifyModUpdates }), Apply: ptrApply(func(s *Settings, on bool) { s.NotifyModUpdates = &on })},
		{Key: "keepDownloadArchives", Get: boolGet(func(s Settings) bool { return s.KeepDownloadArchives }), Apply: boolApply(func(s *Settings, on bool) { s.KeepDownloadArchives = on })},
		{Key: "storeRetentionDays", Get: intGet(func(s Settings) int { return s.StoreRetentionDays }), Apply: intApply(func(s *Settings, n int) { s.StoreRetentionDays = n })},
		{Key: "nxmDefaultProfile", Get: func(s Settings) string { return s.NxmDefaultProfile }, Apply: func(s *Settings, v string) error { s.NxmDefaultProfile = v; return nil }},
		{Key: "defaultModsView", Get: func(s Settings) string { return s.DefaultModsView }, Apply: enumApply(modsViewValues, func(s *Settings, v string) { s.DefaultModsView = v })},
		{Key: "listGroupBy", Get: func(s Settings) string { return s.ListGroupBy }, Apply: func(s *Settings, v string) error { s.ListGroupBy = v; return nil }},
		{Key: "listSortColumn", Get: func(s Settings) string { return s.ListSortColumn }, Apply: func(s *Settings, v string) error { s.ListSortColumn = v; return nil }},
		{Key: "listSortDir", Get: func(s Settings) string { return s.ListSortDir }, Apply: func(s *Settings, v string) error { s.ListSortDir = v; return nil }},
		{Key: "confirmRemovals", Get: ptrGet(func(s Settings) *bool { return s.ConfirmRemovals }), Apply: ptrApply(func(s *Settings, on bool) { s.ConfirmRemovals = &on })},
		{Key: "cosmeticConflicts", Get: func(s Settings) string { return s.CosmeticConflicts }, Apply: enumApply(cosmeticValues, func(s *Settings, v string) { s.CosmeticConflicts = v })},
		{Key: "backgroundBadgeChecks", Get: ptrGet(func(s Settings) *bool { return s.BackgroundBadgeChecks }), Apply: ptrApply(func(s *Settings, on bool) { s.BackgroundBadgeChecks = &on })},
		{Key: "startScreen", Get: func(s Settings) string { return s.StartScreen }, Apply: enumApply(startScreenValues, func(s *Settings, v string) { s.StartScreen = v })},
		{Key: "dates", Get: func(s Settings) string { return s.Dates }, Apply: enumApply(datesValues, func(s *Settings, v string) { s.Dates = v })},
		{Key: "trashRetentionDays", Get: intGet(func(s Settings) int { return s.TrashRetentionDays }), Apply: intApply(func(s *Settings, n int) { s.TrashRetentionDays = n })},
		{Key: "historyEventsKept", Get: intGet(func(s Settings) int { return s.HistoryEventsKept }), Apply: intApply(func(s *Settings, n int) { s.HistoryEventsKept = n })},
		{Key: "notifyDownloadFinished", Get: ptrGet(func(s Settings) *bool { return s.NotifyDownloadFinished }), Apply: ptrApply(func(s *Settings, on bool) { s.NotifyDownloadFinished = &on })},
		{Key: "notifyDownloadFailed", Get: ptrGet(func(s Settings) *bool { return s.NotifyDownloadFailed }), Apply: ptrApply(func(s *Settings, on bool) { s.NotifyDownloadFailed = &on })},
		{Key: "notifyRunCrashed", Get: ptrGet(func(s Settings) *bool { return s.NotifyRunCrashed }), Apply: ptrApply(func(s *Settings, on bool) { s.NotifyRunCrashed = &on })},
		{Key: "density", Get: func(s Settings) string { return s.Density }, Apply: enumApply(densityValues, func(s *Settings, v string) { s.Density = v })},
		{Key: "gridCardSize", Get: func(s Settings) string { return s.GridCardSize }, Apply: enumApply(gridCardValues, func(s *Settings, v string) { s.GridCardSize = v })},
		{Key: "showAuthorOnCards", Get: ptrGet(func(s Settings) *bool { return s.ShowAuthorOnCards }), Apply: ptrApply(func(s *Settings, on bool) { s.ShowAuthorOnCards = &on })},
		{Key: "reduceMotion", Get: func(s Settings) string { return s.ReduceMotion }, Apply: enumApply(reduceMotionValues, func(s *Settings, v string) { s.ReduceMotion = v })},
		{Key: "profileHero", Get: func(s Settings) string { return s.ProfileHero }, Apply: enumApply(heroValues, func(s *Settings, v string) { s.ProfileHero = v })},
		{Key: "enableRequirements", Get: func(s Settings) string { return s.EnableRequirements }, Apply: enumApply(enableReqValues, func(s *Settings, v string) { s.EnableRequirements = v })},
		{Key: "missingRequirements", Get: func(s Settings) string { return s.MissingRequirements }, Apply: enumApply(missingReqValues, func(s *Settings, v string) { s.MissingRequirements = v })},
		{Key: "reuseFomodChoices", Get: ptrGet(func(s Settings) *bool { return s.ReuseFomodChoices }), Apply: ptrApply(func(s *Settings, on bool) { s.ReuseFomodChoices = &on })},
		{Key: "driftChecks", Get: ptrGet(func(s Settings) *bool { return s.DriftChecks }), Apply: ptrApply(func(s *Settings, on bool) { s.DriftChecks = &on })},
		{Key: "smapiBuilds", Get: func(s Settings) string { return s.SmapiBuilds }, Apply: enumApply(smapiBuildsValues, func(s *Settings, v string) { s.SmapiBuilds = v })},
		{Key: "autoInstallMortarUpdates", Get: ptrGet(func(s Settings) *bool { return s.AutoInstallMortarUpdates }), Apply: ptrApply(func(s *Settings, on bool) { s.AutoInstallMortarUpdates = &on })},
		{Key: "autoTrackNexus", Get: boolGet(func(s Settings) bool { return s.AutoTrackNexus }), Apply: boolApply(func(s *Settings, on bool) { s.AutoTrackNexus = on })},
		{Key: "defaultLaunchMethod", Get: func(s Settings) string { return s.DefaultLaunchMethod }, Apply: enumApply(launchMethodValues, func(s *Settings, v string) { s.DefaultLaunchMethod = v })},
		{Key: "showSmapiConsole", Get: ptrGet(func(s Settings) *bool { return s.ShowSmapiConsole }), Apply: ptrApply(func(s *Settings, on bool) { s.ShowSmapiConsole = &on })},
		{Key: "consoleLevel", Get: func(s Settings) string { return s.ConsoleLevel }, Apply: enumApply(consoleLevelValues, func(s *Settings, v string) { s.ConsoleLevel = v })},
		{Key: "consoleTimestamps", Get: ptrGet(func(s Settings) *bool { return s.ConsoleTimestamps }), Apply: ptrApply(func(s *Settings, on bool) { s.ConsoleTimestamps = &on })},
		{Key: "consoleFollow", Get: ptrGet(func(s Settings) *bool { return s.ConsoleFollow }), Apply: ptrApply(func(s *Settings, on bool) { s.ConsoleFollow = &on })},
		{Key: "lanName", Get: func(s Settings) string { return s.LanName }, Apply: func(s *Settings, v string) error { s.LanName = v; return nil }},
		{Key: "lanAutoAcceptSameAccount", Get: boolGet(func(s Settings) bool { return s.LanAutoAcceptSameAccount }), Apply: boolApply(func(s *Settings, on bool) { s.LanAutoAcceptSameAccount = on })},
		{Key: "downloadFolder", Get: func(s Settings) string { return s.DownloadFolder }, Apply: func(s *Settings, v string) error { s.DownloadFolder = v; return nil }},
	}
}

func intGet(fn func(Settings) int) func(Settings) string {
	return func(s Settings) string { return strconv.Itoa(fn(s)) }
}

func boolGet(fn func(Settings) bool) func(Settings) string {
	return func(s Settings) string { return strconv.FormatBool(fn(s)) }
}

func ptrGet(fn func(Settings) *bool) func(Settings) string {
	return func(s Settings) string {
		v := fn(s)
		return strconv.FormatBool(v == nil || *v)
	}
}

func intApply(fn func(*Settings, int)) func(*Settings, string) error {
	return func(s *Settings, raw string) error {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("not an integer: %s", raw)
		}
		fn(s, n)
		return nil
	}
}

func boolApply(fn func(*Settings, bool)) func(*Settings, string) error {
	return func(s *Settings, raw string) error {
		on, err := parseBool(raw)
		if err != nil {
			return err
		}
		fn(s, on)
		return nil
	}
}

func ptrApply(fn func(*Settings, bool)) func(*Settings, string) error {
	return boolApply(fn)
}

func enumApply(allowed []string, fn func(*Settings, string)) func(*Settings, string) error {
	return func(s *Settings, raw string) error {
		if !slices.Contains(allowed, raw) {
			return fmt.Errorf("must be %s, got %q", strings.Join(allowed, ", "), raw)
		}
		fn(s, raw)
		return nil
	}
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "true", "on", "1", "yes":
		return true, nil
	case "false", "off", "0", "no":
		return false, nil
	}
	return false, fmt.Errorf("must be true or false, got %q", raw)
}

func pref(key string) (PrefKey, bool) {
	for _, p := range PrefKeys() {
		if p.Key == key {
			return p, true
		}
	}
	return PrefKey{}, false
}

// Lookup returns the string form of one setting.
func (s Settings) Lookup(key string) (string, error) {
	p, ok := pref(key)
	if !ok {
		return "", fmt.Errorf("unknown setting %q", key)
	}
	return p.Get(s), nil
}

// AllPrefs returns every CLI-visible setting as key/value pairs.
func (s Settings) AllPrefs() [][2]string {
	keys := PrefKeys()
	out := make([][2]string, 0, len(keys))
	for _, p := range keys {
		out = append(out, [2]string{p.Key, p.Get(s)})
	}
	return out
}

// ApplyKey writes one CLI setting onto s.
func ApplyKey(s *Settings, key, value string) error {
	p, ok := pref(key)
	if !ok {
		return fmt.Errorf("unknown setting %q", key)
	}
	return p.Apply(s, value)
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
func (s Settings) AutoEnableRequirements() bool {
	return s.EnableRequirements != EnableReqNever && s.EnableRequirements != EnableReqAsk
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
func (s Settings) ShowConsoleWindow() bool {
	return ToggleOn(s.ShowSmapiConsole)
}

// ArchiveDir is where download zips land; empty means the data folder's downloads directory.
func (s Settings) ArchiveDir() string {
	return strings.TrimSpace(s.DownloadFolder)
}
