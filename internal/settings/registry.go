package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Pref scopes: app (global), source (ScopeSource), loader (ScopeLoader), game (under Settings.Games[gameID]) and profile overrides.
const (
	ScopeApp     = "app"
	ScopeGame    = "game"
	ScopeProfile = "profile"

	TypeBool   = "bool"
	TypeInt    = "int"
	TypeEnum   = "enum"
	TypeString = "string"
)

// GameSettings is the per-game preference block (Stardew Valley today).
type GameSettings struct {
	BackupBeforePlay            string `json:"backupBeforePlay"`
	SaveBackupsKept             int    `json:"saveBackupsKept"`
	SaveBackupHours             int    `json:"saveBackupHours"`
	SaveBackupKeep              int    `json:"saveBackupKeep"`
	UpdateModsBeforePlayDefault bool   `json:"updateModsBeforePlayDefault"`
	RunsKept                    int    `json:"runsKept"`
	ConsoleLogCap               int    `json:"consoleLogCap"`
	NxmDefaultProfile           string `json:"nxmDefaultProfile"`
	CosmeticConflicts           string `json:"cosmeticConflicts"`
	EnableRequirements          string `json:"enableRequirements"`
	MissingRequirements         string `json:"missingRequirements"`
	DefaultLaunchMethod         string `json:"defaultLaunchMethod"`
	SkipPlayCheck               bool   `json:"skipPlayCheck"`
	SkipIntro                   bool   `json:"skipIntro"`
	ConsoleLevel                string `json:"consoleLevel"`
	ConsoleTimestamps           *bool  `json:"consoleTimestamps"`
	ConsoleFollow               *bool  `json:"consoleFollow"`
	BackupLocation              string `json:"backupLocation"`
	ConflictScanDepth           string `json:"conflictScanDepth"`
	OfferNewDownloads           *bool  `json:"offerNewDownloads"`
	// LastDownloadsSeen is the newest archive mtime (ms) in the download folder already offered or skipped.
	LastDownloadsSeen     int64                  `json:"lastDownloadsSeen,omitempty"`
	LastSweepGameVersion  string                 `json:"lastSweepGameVersion,omitempty"`
	LaunchPresetTemplates []LaunchPresetTemplate `json:"launchPresetTemplates,omitempty"`
	ExtraModsFolder       string                 `json:"extraModsFolder,omitempty"`
	ShowDotHiddenMods     bool                   `json:"showDotHiddenMods,omitempty"`
	OldFilesOnUpdate      string                 `json:"oldFilesOnUpdate,omitempty"`
	// SourceOrder is the comma-separated source ids the player prefers, first first; sources it omits follow in
	// catalog order.
	SourceOrder string `json:"sourceOrder,omitempty"`
	// BrowseFilters is Browse's "Show" choice for this game: space-separated row=mode pairs (installed, obsolete,
	// broken; off, gray or hide).
	BrowseFilters string `json:"browseFilters,omitempty"`
	// ListColumns are this game's Mods list columns; empty follows the global default.
	ListColumns []string `json:"listColumns,omitempty"`
}

// PrefSpec is one registry row, served to the CLI and frontend.
type PrefSpec struct {
	Key     string `json:"key"`
	Scope   string `json:"scope"`
	Type    string `json:"type"`
	Default string `json:"default"`
	// GameDefaults holds the games whose default differs from Default, by game id.
	GameDefaults       map[string]string `json:"gameDefaults,omitempty"`
	Min                int               `json:"min,omitempty"`
	Max                int               `json:"max,omitempty"`
	Values             []string          `json:"values,omitempty"`
	ProfileOverridable bool              `json:"profileOverridable,omitempty"`
	// Source is the mod source id a ScopeSource key belongs to, Loader the loader id a ScopeLoader key belongs to.
	Source string `json:"source,omitempty"`
	Loader string `json:"loader,omitempty"`
}

// DefaultFor is the default of a game-scoped key for game, Default for any other.
func (p PrefSpec) DefaultFor(game string) string {
	if v, ok := p.GameDefaults[game]; ok {
		return v
	}
	return p.Default
}

type pref struct {
	spec PrefSpec
	get  func(Settings, string) string
	set  func(*Settings, string, string) error
}

var registry = withDefaults([]pref{
	enumPref("onPlay", ScopeApp, onPlayValues, func(s Settings, _ string) string { return s.OnPlay }, func(s *Settings, _, v string) { s.OnPlay = v }),
	intPref("parallelDownloads", ScopeApp, MinParallelDownloads, MaxParallelDownloads, func(s Settings, _ string) int { return s.ParallelDownloads }, func(s *Settings, _ string, n int) { s.ParallelDownloads = n }),
	intPref("updateCheckIntervalMinutes", ScopeApp, MinUpdateCheckIntervalMinutes, MaxUpdateCheckIntervalMinutes, func(s Settings, _ string) int { return s.UpdateCheckIntervalMinutes }, func(s *Settings, _ string, n int) { s.UpdateCheckIntervalMinutes = n }),
	ptrPref("checkModUpdatesOnStart", ScopeApp, func(s Settings, _ string) *bool { return s.CheckModUpdatesOnStart }, func(s *Settings, _ string, on bool) { s.CheckModUpdatesOnStart = &on }),
	ptrPref("notifyModUpdates", ScopeApp, func(s Settings, _ string) *bool { return s.NotifyModUpdates }, func(s *Settings, _ string, on bool) { s.NotifyModUpdates = &on }),
	enumPref("updateDigest", ScopeApp, updateDigestValues, func(s Settings, _ string) string { return s.UpdateDigest }, func(s *Settings, _, v string) { s.UpdateDigest = v }),
	boolPref("keepDownloadArchives", ScopeApp, func(s Settings, _ string) bool { return s.KeepDownloadArchives }, func(s *Settings, _ string, on bool) { s.KeepDownloadArchives = on }),
	intPref("storeRetentionDays", ScopeApp, MinStoreRetentionDays, MaxStoreRetentionDays, func(s Settings, _ string) int { return s.StoreRetentionDays }, func(s *Settings, _ string, n int) { s.StoreRetentionDays = n }),
	enumPref("defaultModsView", ScopeApp, modsViewValues, func(s Settings, _ string) string { return s.DefaultModsView }, func(s *Settings, _, v string) { s.DefaultModsView = v }),
	strPref("listGroupBy", ScopeApp, func(s Settings, _ string) string { return s.ListGroupBy }, func(s *Settings, _, v string) { s.ListGroupBy = v }),
	strPref("listSortColumn", ScopeApp, func(s Settings, _ string) string { return s.ListSortColumn }, func(s *Settings, _, v string) { s.ListSortColumn = v }),
	strPref("listSortDir", ScopeApp, func(s Settings, _ string) string { return s.ListSortDir }, func(s *Settings, _, v string) { s.ListSortDir = v }),
	ptrPref("confirmRemovals", ScopeApp, func(s Settings, _ string) *bool { return s.ConfirmRemovals }, func(s *Settings, _ string, on bool) { s.ConfirmRemovals = &on }),
	ptrPref("backgroundBadgeChecks", ScopeApp, func(s Settings, _ string) *bool { return s.BackgroundBadgeChecks }, func(s *Settings, _ string, on bool) { s.BackgroundBadgeChecks = &on }),
	enumPref("startScreen", ScopeApp, startScreenValues, func(s Settings, _ string) string { return s.StartScreen }, func(s *Settings, _, v string) { s.StartScreen = v }),
	enumPref("antivirus", ScopeApp, antivirusValues, func(s Settings, _ string) string { return s.Antivirus }, func(s *Settings, _, v string) { s.Antivirus = v }),
	strPref("antivirusSocket", ScopeApp, func(s Settings, _ string) string { return s.AntivirusSocket }, func(s *Settings, _, v string) { s.AntivirusSocket = v }),
	strPref("antivirusCommand", ScopeApp, func(s Settings, _ string) string { return s.AntivirusCommand }, func(s *Settings, _, v string) { s.AntivirusCommand = v }),
	enumPref("dates", ScopeApp, datesValues, func(s Settings, _ string) string { return s.Dates }, func(s *Settings, _, v string) { s.Dates = v }),
	intPref("trashRetentionDays", ScopeApp, MinTrashRetentionDays, MaxTrashRetentionDays, func(s Settings, _ string) int { return s.TrashRetentionDays }, func(s *Settings, _ string, n int) { s.TrashRetentionDays = n }),
	intPref("historyEventsKept", ScopeApp, MinHistoryEventsKept, MaxHistoryEventsKept, func(s Settings, _ string) int { return s.HistoryEventsKept }, func(s *Settings, _ string, n int) { s.HistoryEventsKept = n }),
	ptrPref("notifyDownloadFinished", ScopeApp, func(s Settings, _ string) *bool { return s.NotifyDownloadFinished }, func(s *Settings, _ string, on bool) { s.NotifyDownloadFinished = &on }),
	ptrPref("notifyDownloadFailed", ScopeApp, func(s Settings, _ string) *bool { return s.NotifyDownloadFailed }, func(s *Settings, _ string, on bool) { s.NotifyDownloadFailed = &on }),
	ptrPref("notifyRunCrashed", ScopeApp, func(s Settings, _ string) *bool { return s.NotifyRunCrashed }, func(s *Settings, _ string, on bool) { s.NotifyRunCrashed = &on }),
	ptrPref("desktopDownloadFinished", ScopeApp, func(s Settings, _ string) *bool { return s.DesktopDownloadFinished }, func(s *Settings, _ string, on bool) { s.DesktopDownloadFinished = &on }),
	ptrPref("desktopDownloadFailed", ScopeApp, func(s Settings, _ string) *bool { return s.DesktopDownloadFailed }, func(s *Settings, _ string, on bool) { s.DesktopDownloadFailed = &on }),
	ptrPref("desktopRunCrashed", ScopeApp, func(s Settings, _ string) *bool { return s.DesktopRunCrashed }, func(s *Settings, _ string, on bool) { s.DesktopRunCrashed = &on }),
	ptrPref("desktopModUpdates", ScopeApp, func(s Settings, _ string) *bool { return s.DesktopModUpdates }, func(s *Settings, _ string, on bool) { s.DesktopModUpdates = &on }),
	enumPref("density", ScopeApp, densityValues, func(s Settings, _ string) string { return s.Density }, func(s *Settings, _, v string) { s.Density = v }),
	enumPref("theme", ScopeApp, themeValues, func(s Settings, _ string) string { return s.Theme }, func(s *Settings, _, v string) { s.Theme = v }),
	enumPref("gridCardSize", ScopeApp, gridCardValues, func(s Settings, _ string) string { return s.GridCardSize }, func(s *Settings, _, v string) { s.GridCardSize = v }),
	ptrPref("showAuthorOnCards", ScopeApp, func(s Settings, _ string) *bool { return s.ShowAuthorOnCards }, func(s *Settings, _ string, on bool) { s.ShowAuthorOnCards = &on }),
	enumPref("reduceMotion", ScopeApp, reduceMotionValues, func(s Settings, _ string) string { return s.ReduceMotion }, func(s *Settings, _, v string) { s.ReduceMotion = v }),
	enumPref("profileHero", ScopeApp, heroValues, func(s Settings, _ string) string { return s.ProfileHero }, func(s *Settings, _, v string) { s.ProfileHero = v }),
	ptrPref("reuseFomodChoices", ScopeApp, func(s Settings, _ string) *bool { return s.ReuseFomodChoices }, func(s *Settings, _ string, on bool) { s.ReuseFomodChoices = &on }),
	ptrPref("driftChecks", ScopeApp, func(s Settings, _ string) *bool { return s.DriftChecks }, func(s *Settings, _ string, on bool) { s.DriftChecks = &on }),
	ptrPref("autoInstallMortarUpdates", ScopeApp, func(s Settings, _ string) *bool { return s.AutoInstallMortarUpdates }, func(s *Settings, _ string, on bool) { s.AutoInstallMortarUpdates = &on }),
	sourcePref("nexus", boolPref("autoTrackNexus", ScopeSource, func(s Settings, _ string) bool { return s.AutoTrackNexus }, func(s *Settings, _ string, on bool) { s.AutoTrackNexus = on })),
	intPref("lanPort", ScopeApp, 0, 65535, func(s Settings, _ string) int { return s.LanPort }, func(s *Settings, _ string, n int) { s.LanPort = n }),
	boolPref("lanSharing", ScopeApp, func(s Settings, _ string) bool { return s.LanSharing }, func(s *Settings, _ string, on bool) { s.LanSharing = on }),
	boolPref("lanAllowAnyAddress", ScopeApp, func(s Settings, _ string) bool { return s.LanAllowAnyAddress }, func(s *Settings, _ string, on bool) { s.LanAllowAnyAddress = on }),
	strPref("lanName", ScopeApp, func(s Settings, _ string) string { return s.LanName }, func(s *Settings, _, v string) { s.LanName = v }),
	boolPref("lanAutoAcceptPaired", ScopeApp, func(s Settings, _ string) bool { return s.LanAutoAcceptPaired }, func(s *Settings, _ string, on bool) { s.LanAutoAcceptPaired = on }),
	strPref("downloadFolder", ScopeApp, func(s Settings, _ string) string { return s.DownloadFolder }, func(s *Settings, _, v string) { s.DownloadFolder = v }),
	strPref("syncFolder", ScopeApp, func(s Settings, _ string) string { return s.SyncFolder }, func(s *Settings, _, v string) { s.SyncFolder = v }),
	vortexFolderPref(),
	strPref("watchFolders", ScopeApp, func(s Settings, _ string) string { return s.WatchFolders }, func(s *Settings, _, v string) { s.WatchFolders = v }),
	enumPref("profileOrder", ScopeApp, profileOrderValues, func(s Settings, _ string) string { return s.ProfileOrder }, func(s *Settings, _, v string) { s.ProfileOrder = v }),
	enumPref("autoRetryDownloads", ScopeApp, autoRetryValues, func(s Settings, _ string) string { return s.AutoRetryDownloads }, func(s *Settings, _, v string) { s.AutoRetryDownloads = v }),
	boolPref("pauseDownloadsWhilePlaying", ScopeApp, func(s Settings, _ string) bool { return s.PauseDownloadsWhilePlaying }, func(s *Settings, _ string, on bool) { s.PauseDownloadsWhilePlaying = on }),
	enumPref("sidebarBadges", ScopeApp, sidebarBadgesValues, func(s Settings, _ string) string { return s.SidebarBadges }, func(s *Settings, _, v string) { s.SidebarBadges = v }),
	ptrPref("shareIncludeDisabledMods", ScopeApp, func(s Settings, _ string) *bool { return s.ShareIncludeDisabledMods }, func(s *Settings, _ string, on bool) { s.ShareIncludeDisabledMods = &on }),
	ptrPref("shareIncludeFomodChoices", ScopeApp, func(s Settings, _ string) *bool { return s.ShareIncludeFomodChoices }, func(s *Settings, _ string, on bool) { s.ShareIncludeFomodChoices = &on }),
	ptrPref("shareIncludeNotes", ScopeApp, func(s Settings, _ string) *bool { return s.ShareIncludeNotes }, func(s *Settings, _ string, on bool) { s.ShareIncludeNotes = &on }),
	ptrPref("shareIncludeConfigFiles", ScopeApp, func(s Settings, _ string) *bool { return s.ShareIncludeConfigFiles }, func(s *Settings, _ string, on bool) { s.ShareIncludeConfigFiles = &on }),
	ptrPref("shareIncludeProblemChoices", ScopeApp, func(s Settings, _ string) *bool { return s.ShareIncludeProblemChoices }, func(s *Settings, _ string, on bool) { s.ShareIncludeProblemChoices = &on }),
	sourcePref("nexus", boolPref("verifyNexusMD5", ScopeSource, func(s Settings, _ string) bool { return s.VerifyNexusMD5 }, func(s *Settings, _ string, on bool) { s.VerifyNexusMD5 = on })),
	boolPref("keepInTray", ScopeApp, func(s Settings, _ string) bool { return s.KeepInTray }, func(s *Settings, _ string, on bool) { s.KeepInTray = on }),
	boolPref("includeBetaReleases", ScopeApp, func(s Settings, _ string) bool { return s.IncludeBetaReleases }, func(s *Settings, _ string, on bool) { s.IncludeBetaReleases = on }),
	boolPref("includePrereleaseModVersions", ScopeApp, func(s Settings, _ string) bool { return s.IncludePrereleaseModVersions }, func(s *Settings, _ string, on bool) { s.IncludePrereleaseModVersions = on }),
	ptrPref("askEndorseMods", ScopeApp, func(s Settings, _ string) *bool { return s.AskEndorseMods }, func(s *Settings, _ string, on bool) { s.AskEndorseMods = &on }),
	listColumnsPref(),
	boolPref("launchAtLogin", ScopeApp, func(s Settings, _ string) bool { return s.LaunchAtLogin }, func(s *Settings, _ string, on bool) {
		s.LaunchAtLogin = on
	}),
	boolPref("showAdultContent", ScopeApp, func(s Settings, _ string) bool { return s.ShowAdultContent }, func(s *Settings, _ string, on bool) { s.ShowAdultContent = on }),
	boolPref("startMinimised", ScopeApp, func(s Settings, _ string) bool { return s.StartMinimised }, func(s *Settings, _ string, on bool) { s.StartMinimised = on }),
	boolPref("rememberWindow", ScopeApp, func(s Settings, _ string) bool { return s.RememberWindow }, func(s *Settings, _ string, on bool) { s.RememberWindow = on }),
	enumPref("extensionConnection", ScopeApp, extensionConnectionValues, func(s Settings, _ string) string { return s.ExtensionConnection }, func(s *Settings, _, v string) { s.ExtensionConnection = v }),

	overridable(enumPref("backupBeforePlay", ScopeGame, backupBeforePlayValues, func(s Settings, g string) string { return s.GamePrefs(g).BackupBeforePlay }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.BackupBeforePlay = v; putGame(s, g, gp) })),
	overridable(intPref("saveBackupsKept", ScopeGame, MinSaveBackupsKept, MaxSaveBackupsKept, func(s Settings, g string) int { return s.GamePrefs(g).SaveBackupsKept }, func(s *Settings, g string, n int) { gp := s.GamePrefs(g); gp.SaveBackupsKept = n; putGame(s, g, gp) })),
	overridable(boolPref("updateModsBeforePlayDefault", ScopeGame, func(s Settings, g string) bool { return s.GamePrefs(g).UpdateModsBeforePlayDefault }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.UpdateModsBeforePlayDefault = on
		putGame(s, g, gp)
	})),
	intPref("saveBackupHours", ScopeGame, MinSaveBackupHours, MaxSaveBackupHours, func(s Settings, g string) int { return s.GamePrefs(g).SaveBackupHours }, func(s *Settings, g string, n int) { gp := s.GamePrefs(g); gp.SaveBackupHours = n; putGame(s, g, gp) }),
	intPref("saveBackupKeep", ScopeGame, MinSaveBackupsKept, MaxSaveBackupsKept, func(s Settings, g string) int { return s.GamePrefs(g).SaveBackupKeep }, func(s *Settings, g string, n int) { gp := s.GamePrefs(g); gp.SaveBackupKeep = n; putGame(s, g, gp) }),
	intPref("runsKept", ScopeGame, MinRunsKept, MaxRunsKept, func(s Settings, g string) int { return s.GamePrefs(g).RunsKept }, func(s *Settings, g string, n int) { gp := s.GamePrefs(g); gp.RunsKept = n; putGame(s, g, gp) }),
	intPref("consoleLogCap", ScopeGame, MinConsoleLogCap, MaxConsoleLogCap, func(s Settings, g string) int { return s.GamePrefs(g).ConsoleLogCap }, func(s *Settings, g string, n int) { gp := s.GamePrefs(g); gp.ConsoleLogCap = n; putGame(s, g, gp) }),
	strPref("nxmDefaultProfile", ScopeGame, func(s Settings, g string) string { return s.GamePrefs(g).NxmDefaultProfile }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.NxmDefaultProfile = v; putGame(s, g, gp) }),
	enumPref("cosmeticConflicts", ScopeGame, cosmeticValues, func(s Settings, g string) string { return s.GamePrefs(g).CosmeticConflicts }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.CosmeticConflicts = v; putGame(s, g, gp) }),
	enumPref("enableRequirements", ScopeGame, enableReqValues, func(s Settings, g string) string { return s.GamePrefs(g).EnableRequirements }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.EnableRequirements = v; putGame(s, g, gp) }),
	enumPref("missingRequirements", ScopeGame, missingReqValues, func(s Settings, g string) string { return s.GamePrefs(g).MissingRequirements }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.MissingRequirements = v; putGame(s, g, gp) }),
	loaderPref("smapi", enumPref("smapiBuilds", ScopeLoader, smapiBuildsValues, func(s Settings, _ string) string { return s.SmapiBuilds }, func(s *Settings, _, v string) { s.SmapiBuilds = v })),
	loaderPref("smapi", pinPref("smapiPin", "smapi")),
	loaderPref("bepinex5", pinPref("bepinex5Pin", "bepinex5")),
	overridable(enumPref("defaultLaunchMethod", ScopeGame, launchMethodValues, func(s Settings, g string) string { return s.GamePrefs(g).DefaultLaunchMethod }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.DefaultLaunchMethod = v; putGame(s, g, gp) })),
	overridable(loaderPref("smapi", ptrPref("showSmapiConsole", ScopeLoader, func(s Settings, _ string) *bool { return s.ShowSmapiConsole }, func(s *Settings, _ string, on bool) { s.ShowSmapiConsole = &on }))),
	overridable(boolPref("skipPlayCheck", ScopeGame, func(s Settings, g string) bool { return s.GamePrefs(g).SkipPlayCheck }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.SkipPlayCheck = on
		putGame(s, g, gp)
	})),
	overridable(boolPref("skipIntro", ScopeGame, func(s Settings, g string) bool { return s.GamePrefs(g).SkipIntro }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.SkipIntro = on
		putGame(s, g, gp)
	})),
	enumPref("consoleLevel", ScopeGame, consoleLevelValues, func(s Settings, g string) string { return s.GamePrefs(g).ConsoleLevel }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.ConsoleLevel = v; putGame(s, g, gp) }),
	ptrPref("consoleTimestamps", ScopeGame, func(s Settings, g string) *bool { return s.GamePrefs(g).ConsoleTimestamps }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.ConsoleTimestamps = &on
		putGame(s, g, gp)
	}),
	ptrPref("consoleFollow", ScopeGame, func(s Settings, g string) *bool { return s.GamePrefs(g).ConsoleFollow }, func(s *Settings, g string, on bool) { gp := s.GamePrefs(g); gp.ConsoleFollow = &on; putGame(s, g, gp) }),
	strPref("backupLocation", ScopeGame, func(s Settings, g string) string { return s.GamePrefs(g).BackupLocation }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.BackupLocation = v; putGame(s, g, gp) }),
	enumPref("conflictScanDepth", ScopeGame, conflictScanValues, func(s Settings, g string) string { return s.GamePrefs(g).ConflictScanDepth }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.ConflictScanDepth = v; putGame(s, g, gp) }),
	ptrPref("offerNewDownloads", ScopeGame, func(s Settings, g string) *bool { return s.GamePrefs(g).OfferNewDownloads }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.OfferNewDownloads = &on
		putGame(s, g, gp)
	}),
	strPref("extraModsFolder", ScopeGame, func(s Settings, g string) string { return s.GamePrefs(g).ExtraModsFolder }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.ExtraModsFolder = v; putGame(s, g, gp) }),
	boolPref("showDotHiddenMods", ScopeGame, func(s Settings, g string) bool { return s.GamePrefs(g).ShowDotHiddenMods }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.ShowDotHiddenMods = on
		putGame(s, g, gp)
	}),
	strPref("browseFilters", ScopeGame, func(s Settings, g string) string { return s.GamePrefs(g).BrowseFilters }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.BrowseFilters = v; putGame(s, g, gp) }),
	strPref("sourceOrder", ScopeGame, func(s Settings, g string) string { return s.GamePrefs(g).SourceOrder }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.SourceOrder = v; putGame(s, g, gp) }),
	enumPref("oldFilesOnUpdate", ScopeGame, oldFilesValues, func(s Settings, g string) string { return s.GamePrefs(g).OldFilesOnUpdate }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.OldFilesOnUpdate = v; putGame(s, g, gp) }),
})

// gameLaunchDefaults are the games whose default launch method is not Steam's.
var gameLaunchDefaults = map[string]string{"stardew": LaunchDirect}

// withDefaults fills each row's default from Defaults(), so the registry cannot drift from it.
func withDefaults(rows []pref) []pref {
	d := Defaults()
	for i := range rows {
		p := &rows[i]
		p.spec.Default = p.get(d, "")
		if p.spec.Scope != ScopeGame {
			continue
		}
		for game := range gameLaunchDefaults {
			if v := p.get(d, game); v != p.spec.Default {
				if p.spec.GameDefaults == nil {
					p.spec.GameDefaults = map[string]string{}
				}
				p.spec.GameDefaults[game] = v
			}
		}
	}
	return rows
}

// defaultGameSettingsFor is defaultGameSettings with the game's own launch default.
func defaultGameSettingsFor(game string) GameSettings {
	g := defaultGameSettings()
	if m, ok := gameLaunchDefaults[game]; ok {
		g.DefaultLaunchMethod = m
	}
	return g
}

func defaultGameSettings() GameSettings {
	return GameSettings{
		BackupBeforePlay:            BackupBeforePlayChanged,
		SaveBackupsKept:             DefaultSaveBackupsKept,
		SaveBackupKeep:              DefaultSaveBackupKeep,
		UpdateModsBeforePlayDefault: false,
		RunsKept:                    DefaultRunsKept,
		ConsoleLogCap:               DefaultConsoleLogCap,
		NxmDefaultProfile:           "",
		CosmeticConflicts:           CosmeticCollapsed,
		EnableRequirements:          EnableReqAlways,
		MissingRequirements:         MissingReqAsk,
		DefaultLaunchMethod:         LaunchSteam,
		ConsoleLevel:                ConsoleLevelWarn,
		ConsoleTimestamps:           off(),
		ConsoleFollow:               on(),
		BackupLocation:              "",
		ConflictScanDepth:           ConflictScanFull,
		OfferNewDownloads:           on(),
		OldFilesOnUpdate:            OldFilesAsk,
	}
}

// GamePrefs returns stored game prefs merged with defaults.
func (s Settings) GamePrefs(gameID string) GameSettings {
	out := defaultGameSettingsFor(gameID)
	if gameID == "" || s.Games == nil {
		return out
	}
	got, ok := s.Games[gameID]
	if !ok || got == nil {
		return out
	}
	mergeGame(&out, *got)
	return out
}

func putGame(s *Settings, gameID string, g GameSettings) {
	if s.Games == nil {
		s.Games = map[string]*GameSettings{}
	}
	cp := g
	s.Games[gameID] = &cp
}

func mergeGame(dst *GameSettings, src GameSettings) {
	if src.SourceOrder != "" {
		dst.SourceOrder = src.SourceOrder
	}
	if src.BrowseFilters != "" {
		dst.BrowseFilters = src.BrowseFilters
	}
	if src.BackupBeforePlay != "" {
		dst.BackupBeforePlay = src.BackupBeforePlay
	}
	if src.SaveBackupsKept != 0 {
		dst.SaveBackupsKept = src.SaveBackupsKept
	}
	dst.SaveBackupHours = src.SaveBackupHours
	if src.SaveBackupKeep != 0 {
		dst.SaveBackupKeep = src.SaveBackupKeep
	}
	dst.UpdateModsBeforePlayDefault = src.UpdateModsBeforePlayDefault
	dst.SkipPlayCheck = src.SkipPlayCheck
	dst.SkipIntro = src.SkipIntro
	if src.RunsKept != 0 {
		dst.RunsKept = src.RunsKept
	}
	if src.ConsoleLogCap != 0 {
		dst.ConsoleLogCap = src.ConsoleLogCap
	}
	dst.NxmDefaultProfile = src.NxmDefaultProfile
	if src.CosmeticConflicts != "" {
		dst.CosmeticConflicts = src.CosmeticConflicts
	}
	if src.EnableRequirements != "" {
		dst.EnableRequirements = src.EnableRequirements
	}
	if src.MissingRequirements != "" {
		dst.MissingRequirements = src.MissingRequirements
	}
	if src.DefaultLaunchMethod != "" {
		dst.DefaultLaunchMethod = src.DefaultLaunchMethod
	}
	if src.ConsoleLevel != "" {
		dst.ConsoleLevel = src.ConsoleLevel
	}
	if src.ConsoleTimestamps != nil {
		dst.ConsoleTimestamps = src.ConsoleTimestamps
	}
	if src.ConsoleFollow != nil {
		dst.ConsoleFollow = src.ConsoleFollow
	}
	dst.BackupLocation = src.BackupLocation
	if src.ConflictScanDepth != "" {
		dst.ConflictScanDepth = src.ConflictScanDepth
	}
	if src.OfferNewDownloads != nil {
		dst.OfferNewDownloads = src.OfferNewDownloads
	}
	if src.LastDownloadsSeen != 0 {
		dst.LastDownloadsSeen = src.LastDownloadsSeen
	}
	if src.LastSweepGameVersion != "" {
		dst.LastSweepGameVersion = src.LastSweepGameVersion
	}
	dst.LaunchPresetTemplates = append([]LaunchPresetTemplate(nil), src.LaunchPresetTemplates...)
	dst.ExtraModsFolder = src.ExtraModsFolder
	dst.ShowDotHiddenMods = src.ShowDotHiddenMods
	dst.ListColumns = slices.Clone(src.ListColumns)
	if src.OldFilesOnUpdate != "" {
		dst.OldFilesOnUpdate = src.OldFilesOnUpdate
	}
}

func normalizeGame(id string, g *GameSettings) {
	d := defaultGameSettingsFor(id)
	if len(g.ListColumns) > 0 {
		g.ListColumns = sanitizeListColumns(g.ListColumns)
	}
	if !slices.Contains(backupBeforePlayValues, g.BackupBeforePlay) {
		g.BackupBeforePlay = d.BackupBeforePlay
	}
	if g.SaveBackupsKept < MinSaveBackupsKept || g.SaveBackupsKept > MaxSaveBackupsKept {
		g.SaveBackupsKept = d.SaveBackupsKept
	}
	if g.SaveBackupHours < MinSaveBackupHours || g.SaveBackupHours > MaxSaveBackupHours {
		g.SaveBackupHours = d.SaveBackupHours
	}
	if g.SaveBackupKeep < MinSaveBackupsKept || g.SaveBackupKeep > MaxSaveBackupsKept {
		g.SaveBackupKeep = d.SaveBackupKeep
	}
	if g.RunsKept < MinRunsKept || g.RunsKept > MaxRunsKept {
		g.RunsKept = d.RunsKept
	}
	if g.ConsoleLogCap < MinConsoleLogCap || g.ConsoleLogCap > MaxConsoleLogCap {
		g.ConsoleLogCap = d.ConsoleLogCap
	}
	if !slices.Contains(cosmeticValues, g.CosmeticConflicts) {
		g.CosmeticConflicts = d.CosmeticConflicts
	}
	if !slices.Contains(enableReqValues, g.EnableRequirements) {
		g.EnableRequirements = d.EnableRequirements
	}
	if !slices.Contains(missingReqValues, g.MissingRequirements) {
		g.MissingRequirements = d.MissingRequirements
	}
	if !slices.Contains(launchMethodValues, g.DefaultLaunchMethod) {
		g.DefaultLaunchMethod = d.DefaultLaunchMethod
	}
	if !slices.Contains(consoleLevelValues, g.ConsoleLevel) {
		g.ConsoleLevel = d.ConsoleLevel
	}
	if g.ConsoleTimestamps == nil {
		g.ConsoleTimestamps = d.ConsoleTimestamps
	}
	if g.ConsoleFollow == nil {
		g.ConsoleFollow = d.ConsoleFollow
	}
	if !slices.Contains(conflictScanValues, g.ConflictScanDepth) {
		g.ConflictScanDepth = d.ConflictScanDepth
	}
	if g.OfferNewDownloads == nil {
		g.OfferNewDownloads = d.OfferNewDownloads
	}
	if !slices.Contains(oldFilesValues, g.OldFilesOnUpdate) {
		g.OldFilesOnUpdate = d.OldFilesOnUpdate
	}
}

func validateGame(g GameSettings) error {
	if !slices.Contains(backupBeforePlayValues, g.BackupBeforePlay) {
		return fmt.Errorf("backup before play must be changed, always or never, got %q", g.BackupBeforePlay)
	}
	if g.SaveBackupsKept < MinSaveBackupsKept || g.SaveBackupsKept > MaxSaveBackupsKept {
		return fmt.Errorf("launch backups kept must be %d to %d, got %d", MinSaveBackupsKept, MaxSaveBackupsKept, g.SaveBackupsKept)
	}
	if g.SaveBackupHours < MinSaveBackupHours || g.SaveBackupHours > MaxSaveBackupHours {
		return fmt.Errorf("save backup hours must be %d to %d, got %d", MinSaveBackupHours, MaxSaveBackupHours, g.SaveBackupHours)
	}
	if g.SaveBackupKeep < MinSaveBackupsKept || g.SaveBackupKeep > MaxSaveBackupsKept {
		return fmt.Errorf("scheduled save backups kept must be %d to %d, got %d", MinSaveBackupsKept, MaxSaveBackupsKept, g.SaveBackupKeep)
	}
	if g.RunsKept < MinRunsKept || g.RunsKept > MaxRunsKept {
		return fmt.Errorf("runs kept must be %d to %d, got %d", MinRunsKept, MaxRunsKept, g.RunsKept)
	}
	if g.ConsoleLogCap < MinConsoleLogCap || g.ConsoleLogCap > MaxConsoleLogCap {
		return fmt.Errorf("console log cap must be %d to %d, got %d", MinConsoleLogCap, MaxConsoleLogCap, g.ConsoleLogCap)
	}
	if !slices.Contains(cosmeticValues, g.CosmeticConflicts) {
		return fmt.Errorf("cosmetic conflicts must be collapsed, expanded or hidden, got %q", g.CosmeticConflicts)
	}
	if !slices.Contains(enableReqValues, g.EnableRequirements) {
		return fmt.Errorf("enable requirements must be always, ask or never, got %q", g.EnableRequirements)
	}
	if !slices.Contains(missingReqValues, g.MissingRequirements) {
		return fmt.Errorf("missing requirements must be ask, autodownload or never, got %q", g.MissingRequirements)
	}
	if !slices.Contains(launchMethodValues, g.DefaultLaunchMethod) {
		return fmt.Errorf("default launch method must be steam or direct, got %q", g.DefaultLaunchMethod)
	}
	if !slices.Contains(consoleLevelValues, g.ConsoleLevel) {
		return fmt.Errorf("console level must be trace, debug, info, warn or error, got %q", g.ConsoleLevel)
	}
	if !slices.Contains(conflictScanValues, g.ConflictScanDepth) {
		return fmt.Errorf("conflict scan depth must be full or skipImages, got %q", g.ConflictScanDepth)
	}
	if !slices.Contains(oldFilesValues, g.OldFilesOnUpdate) {
		return fmt.Errorf("old files on update must be ask, delete or keep, got %q", g.OldFilesOnUpdate)
	}
	return nil
}

// PrefSpecs is the descriptor the frontend renders.
func PrefSpecs() []PrefSpec {
	out := make([]PrefSpec, len(registry))
	for i, p := range registry {
		out[i] = p.spec
	}
	return out
}

func lookupPref(key string) (pref, bool) {
	for _, p := range registry {
		if p.spec.Key == key {
			return p, true
		}
	}
	return pref{}, false
}

func requireGame(p pref, game string) error {
	if p.spec.Scope != ScopeGame {
		return nil
	}
	if strings.TrimSpace(game) == "" {
		return fmt.Errorf("setting %q is per-game; pass --game", p.spec.Key)
	}
	return nil
}

// Lookup returns the string form of one setting. Game-scoped keys need game.
func (s Settings) Lookup(key string) (string, error) {
	return s.LookupGame(key, "")
}

// LookupGame returns one setting, using game for ScopeGame keys.
func (s Settings) LookupGame(key, game string) (string, error) {
	p, ok := lookupPref(key)
	if !ok {
		return "", fmt.Errorf("unknown setting %q", key)
	}
	if err := requireGame(p, game); err != nil {
		return "", err
	}
	if p.spec.Scope != ScopeGame {
		game = ""
	}
	return p.get(s, game), nil
}

// AllPrefsGame returns app keys, plus game keys when game is set.
func (s Settings) AllPrefsGame(game string) [][2]string {
	out := make([][2]string, 0, len(registry))
	for _, p := range registry {
		if p.spec.Scope == ScopeGame && game == "" {
			continue
		}
		g := game
		if p.spec.Scope != ScopeGame {
			g = ""
		}
		out = append(out, [2]string{p.spec.Key, p.get(s, g)})
	}
	return out
}

// ApplyKeyGame writes one CLI setting onto s.
func ApplyKeyGame(s *Settings, key, value, game string) error {
	p, ok := lookupPref(key)
	if !ok {
		return fmt.Errorf("unknown setting %q", key)
	}
	if err := requireGame(p, game); err != nil {
		return err
	}
	if p.spec.Scope != ScopeGame {
		game = ""
	}
	if p.spec.Type == TypeInt {
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("not an integer: %s", value)
		}
		if p.spec.Min != 0 || p.spec.Max != 0 {
			if n < p.spec.Min || n > p.spec.Max {
				return fmt.Errorf("%s must be %d to %d, got %d", p.spec.Key, p.spec.Min, p.spec.Max, n)
			}
		}
	}
	if p.spec.Type == TypeEnum && len(p.spec.Values) > 0 && !slices.Contains(p.spec.Values, value) {
		return fmt.Errorf("must be %s, got %q", strings.Join(p.spec.Values, ", "), value)
	}
	return p.set(s, game, value)
}

func enumPref(key, scope string, values []string, get func(Settings, string) string, set func(*Settings, string, string)) pref {
	return pref{
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeEnum, Values: values},
		get:  get,
		set: func(s *Settings, game, raw string) error {
			if !slices.Contains(values, raw) {
				return fmt.Errorf("must be %s, got %q", strings.Join(values, ", "), raw)
			}
			set(s, game, raw)
			return nil
		},
	}
}

func intPref(key, scope string, lo, hi int, get func(Settings, string) int, set func(*Settings, string, int)) pref {
	return pref{
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeInt, Min: lo, Max: hi},
		get:  func(s Settings, g string) string { return strconv.Itoa(get(s, g)) },
		set: func(s *Settings, game, raw string) error {
			n, err := strconv.Atoi(raw)
			if err != nil {
				return fmt.Errorf("not an integer: %s", raw)
			}
			if n < lo || n > hi {
				return fmt.Errorf("%s must be %d to %d, got %d", key, lo, hi, n)
			}
			set(s, game, n)
			return nil
		},
	}
}

func boolPref(key, scope string, get func(Settings, string) bool, set func(*Settings, string, bool)) pref {
	return pref{
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeBool},
		get:  func(s Settings, g string) string { return strconv.FormatBool(get(s, g)) },
		set: func(s *Settings, game, raw string) error {
			on, err := parseBool(raw)
			if err != nil {
				return err
			}
			set(s, game, on)
			return nil
		},
	}
}

func ptrPref(key, scope string, get func(Settings, string) *bool, set func(*Settings, string, bool)) pref {
	return pref{
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeBool},
		get: func(s Settings, g string) string {
			v := get(s, g)
			if v == nil {
				v = get(Defaults(), g)
			}
			return strconv.FormatBool(*v)
		},
		set: func(s *Settings, game, raw string) error {
			on, err := parseBool(raw)
			if err != nil {
				return err
			}
			set(s, game, on)
			return nil
		},
	}
}

func strPref(key, scope string, get func(Settings, string) string, set func(*Settings, string, string)) pref {
	return pref{
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeString},
		get:  get,
		set: func(s *Settings, game, raw string) error {
			set(s, game, raw)
			return nil
		},
	}
}

// vortexFolderPref accepts only an existing folder holding Vortex's state.v2, or "" to clear it.
func vortexFolderPref() pref {
	return pref{
		spec: PrefSpec{Key: "vortexFolder", Scope: ScopeApp, Type: TypeString},
		get:  func(s Settings, _ string) string { return s.VortexFolder },
		set: func(s *Settings, _, raw string) error {
			if raw != "" {
				if info, err := os.Stat(filepath.Join(raw, "state.v2")); err != nil || !info.IsDir() {
					return fmt.Errorf("%s has no Vortex data (state.v2)", raw)
				}
			}
			s.VortexFolder = raw
			return nil
		},
	}
}

// listColumnsPref is the global Mods list columns as a comma-separated list of column ids.
func listColumnsPref() pref {
	return pref{
		spec: PrefSpec{Key: "listColumns", Scope: ScopeApp, Type: TypeString},
		get:  func(s Settings, _ string) string { return strings.Join(s.ListColumns, ",") },
		set: func(s *Settings, _, raw string) error {
			ids := strings.Split(raw, ",")
			for _, id := range ids {
				if !knownListColumn(id) {
					return fmt.Errorf("unknown list column %q", id)
				}
			}
			s.ListColumns = sanitizeListColumns(ids)
			return nil
		},
	}
}
