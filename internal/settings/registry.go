package settings

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Pref scopes. Game keys live under Settings.Games[gameID].
const (
	ScopeApp     = "app"
	ScopeGame    = "game"
	ScopeProfile = "profile"

	TypeBool   = "bool"
	TypeInt    = "int"
	TypeEnum   = "enum"
	TypeString = "string"

	GameStardew = "stardew"
)

// GameSettings is the per-game preference block (Stardew Valley today).
type GameSettings struct {
	BackupBeforePlay            string `json:"backupBeforePlay"`
	LaunchBackupsKept           int    `json:"launchBackupsKept"`
	UpdateModsBeforePlayDefault bool   `json:"updateModsBeforePlayDefault"`
	RunsKept                    int    `json:"runsKept"`
	ConsoleLogCap               int    `json:"consoleLogCap"`
	NxmDefaultProfile           string `json:"nxmDefaultProfile"`
	CosmeticConflicts           string `json:"cosmeticConflicts"`
	EnableRequirements          string `json:"enableRequirements"`
	MissingRequirements         string `json:"missingRequirements"`
	SmapiBuilds                 string `json:"smapiBuilds"`
	DefaultLaunchMethod         string `json:"defaultLaunchMethod"`
	ShowSmapiConsole            *bool  `json:"showSmapiConsole"`
	SkipPlayCheck               bool   `json:"skipPlayCheck"`
	ConsoleLevel                string `json:"consoleLevel"`
	ConsoleTimestamps           *bool  `json:"consoleTimestamps"`
	ConsoleFollow               *bool  `json:"consoleFollow"`
}

// PrefSpec is one registry row, served to the CLI and frontend.
type PrefSpec struct {
	Key                string   `json:"key"`
	Scope              string   `json:"scope"`
	Type               string   `json:"type"`
	Default            string   `json:"default"`
	Min                int      `json:"min,omitempty"`
	Max                int      `json:"max,omitempty"`
	Values             []string `json:"values,omitempty"`
	LabelKey           string   `json:"labelKey"`
	ProfileOverridable bool     `json:"profileOverridable,omitempty"`
}

type pref struct {
	spec PrefSpec
	get  func(Settings, string) string
	set  func(*Settings, string, string) error
}

var registry = []pref{
	enumPref("onPlay", ScopeApp, OnPlayStay, onPlayValues, func(s Settings, _ string) string { return s.OnPlay }, func(s *Settings, _, v string) { s.OnPlay = v }),
	intPref("parallelDownloads", ScopeApp, DefaultParallelDownloads, MinParallelDownloads, MaxParallelDownloads, func(s Settings, _ string) int { return s.ParallelDownloads }, func(s *Settings, _ string, n int) { s.ParallelDownloads = n }),
	intPref("updateCheckIntervalMinutes", ScopeApp, DefaultUpdateCheckIntervalMinutes, MinUpdateCheckIntervalMinutes, MaxUpdateCheckIntervalMinutes, func(s Settings, _ string) int { return s.UpdateCheckIntervalMinutes }, func(s *Settings, _ string, n int) { s.UpdateCheckIntervalMinutes = n }),
	ptrPref("checkModUpdatesOnStart", ScopeApp, true, func(s Settings, _ string) *bool { return s.CheckModUpdatesOnStart }, func(s *Settings, _ string, on bool) { s.CheckModUpdatesOnStart = &on }),
	ptrPref("notifyModUpdates", ScopeApp, false, func(s Settings, _ string) *bool { return s.NotifyModUpdates }, func(s *Settings, _ string, on bool) { s.NotifyModUpdates = &on }),
	boolPref("keepDownloadArchives", ScopeApp, func(s Settings, _ string) bool { return s.KeepDownloadArchives }, func(s *Settings, _ string, on bool) { s.KeepDownloadArchives = on }),
	intPref("storeRetentionDays", ScopeApp, DefaultStoreRetentionDays, MinStoreRetentionDays, MaxStoreRetentionDays, func(s Settings, _ string) int { return s.StoreRetentionDays }, func(s *Settings, _ string, n int) { s.StoreRetentionDays = n }),
	enumPref("defaultModsView", ScopeApp, ModsViewGrid, modsViewValues, func(s Settings, _ string) string { return s.DefaultModsView }, func(s *Settings, _, v string) { s.DefaultModsView = v }),
	strPref("listGroupBy", ScopeApp, func(s Settings, _ string) string { return s.ListGroupBy }, func(s *Settings, _, v string) { s.ListGroupBy = v }),
	strPref("listSortColumn", ScopeApp, func(s Settings, _ string) string { return s.ListSortColumn }, func(s *Settings, _, v string) { s.ListSortColumn = v }),
	strPref("listSortDir", ScopeApp, func(s Settings, _ string) string { return s.ListSortDir }, func(s *Settings, _, v string) { s.ListSortDir = v }),
	ptrPref("confirmRemovals", ScopeApp, true, func(s Settings, _ string) *bool { return s.ConfirmRemovals }, func(s *Settings, _ string, on bool) { s.ConfirmRemovals = &on }),
	ptrPref("backgroundBadgeChecks", ScopeApp, true, func(s Settings, _ string) *bool { return s.BackgroundBadgeChecks }, func(s *Settings, _ string, on bool) { s.BackgroundBadgeChecks = &on }),
	enumPref("startScreen", ScopeApp, StartScreenLast, startScreenValues, func(s Settings, _ string) string { return s.StartScreen }, func(s *Settings, _, v string) { s.StartScreen = v }),
	enumPref("dates", ScopeApp, DatesRelative, datesValues, func(s Settings, _ string) string { return s.Dates }, func(s *Settings, _, v string) { s.Dates = v }),
	intPref("trashRetentionDays", ScopeApp, DefaultTrashRetentionDays, MinTrashRetentionDays, MaxTrashRetentionDays, func(s Settings, _ string) int { return s.TrashRetentionDays }, func(s *Settings, _ string, n int) { s.TrashRetentionDays = n }),
	intPref("historyEventsKept", ScopeApp, DefaultHistoryEventsKept, MinHistoryEventsKept, MaxHistoryEventsKept, func(s Settings, _ string) int { return s.HistoryEventsKept }, func(s *Settings, _ string, n int) { s.HistoryEventsKept = n }),
	ptrPref("notifyDownloadFinished", ScopeApp, true, func(s Settings, _ string) *bool { return s.NotifyDownloadFinished }, func(s *Settings, _ string, on bool) { s.NotifyDownloadFinished = &on }),
	ptrPref("notifyDownloadFailed", ScopeApp, true, func(s Settings, _ string) *bool { return s.NotifyDownloadFailed }, func(s *Settings, _ string, on bool) { s.NotifyDownloadFailed = &on }),
	ptrPref("notifyRunCrashed", ScopeApp, true, func(s Settings, _ string) *bool { return s.NotifyRunCrashed }, func(s *Settings, _ string, on bool) { s.NotifyRunCrashed = &on }),
	enumPref("density", ScopeApp, DensityComfortable, densityValues, func(s Settings, _ string) string { return s.Density }, func(s *Settings, _, v string) { s.Density = v }),
	enumPref("gridCardSize", ScopeApp, GridCardMedium, gridCardValues, func(s Settings, _ string) string { return s.GridCardSize }, func(s *Settings, _, v string) { s.GridCardSize = v }),
	ptrPref("showAuthorOnCards", ScopeApp, true, func(s Settings, _ string) *bool { return s.ShowAuthorOnCards }, func(s *Settings, _ string, on bool) { s.ShowAuthorOnCards = &on }),
	enumPref("reduceMotion", ScopeApp, ReduceMotionSystem, reduceMotionValues, func(s Settings, _ string) string { return s.ReduceMotion }, func(s *Settings, _, v string) { s.ReduceMotion = v }),
	enumPref("profileHero", ScopeApp, HeroFull, heroValues, func(s Settings, _ string) string { return s.ProfileHero }, func(s *Settings, _, v string) { s.ProfileHero = v }),
	ptrPref("reuseFomodChoices", ScopeApp, true, func(s Settings, _ string) *bool { return s.ReuseFomodChoices }, func(s *Settings, _ string, on bool) { s.ReuseFomodChoices = &on }),
	ptrPref("driftChecks", ScopeApp, true, func(s Settings, _ string) *bool { return s.DriftChecks }, func(s *Settings, _ string, on bool) { s.DriftChecks = &on }),
	ptrPref("autoInstallMortarUpdates", ScopeApp, true, func(s Settings, _ string) *bool { return s.AutoInstallMortarUpdates }, func(s *Settings, _ string, on bool) { s.AutoInstallMortarUpdates = &on }),
	boolPref("autoTrackNexus", ScopeApp, func(s Settings, _ string) bool { return s.AutoTrackNexus }, func(s *Settings, _ string, on bool) { s.AutoTrackNexus = on }),
	strPref("lanName", ScopeApp, func(s Settings, _ string) string { return s.LanName }, func(s *Settings, _, v string) { s.LanName = v }),
	boolPref("lanAutoAcceptSameAccount", ScopeApp, func(s Settings, _ string) bool { return s.LanAutoAcceptSameAccount }, func(s *Settings, _ string, on bool) { s.LanAutoAcceptSameAccount = on }),
	strPref("downloadFolder", ScopeApp, func(s Settings, _ string) string { return s.DownloadFolder }, func(s *Settings, _, v string) { s.DownloadFolder = v }),

	overridable(enumPref("backupBeforePlay", ScopeGame, BackupBeforePlayChanged, backupBeforePlayValues, func(s Settings, g string) string { return s.GamePrefs(g).BackupBeforePlay }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.BackupBeforePlay = v; putGame(s, g, gp) })),
	overridable(intPref("launchBackupsKept", ScopeGame, DefaultLaunchBackupsKept, MinLaunchBackupsKept, MaxLaunchBackupsKept, func(s Settings, g string) int { return s.GamePrefs(g).LaunchBackupsKept }, func(s *Settings, g string, n int) { gp := s.GamePrefs(g); gp.LaunchBackupsKept = n; putGame(s, g, gp) })),
	overridable(boolPref("updateModsBeforePlayDefault", ScopeGame, func(s Settings, g string) bool { return s.GamePrefs(g).UpdateModsBeforePlayDefault }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.UpdateModsBeforePlayDefault = on
		putGame(s, g, gp)
	})),
	intPref("runsKept", ScopeGame, DefaultRunsKept, MinRunsKept, MaxRunsKept, func(s Settings, g string) int { return s.GamePrefs(g).RunsKept }, func(s *Settings, g string, n int) { gp := s.GamePrefs(g); gp.RunsKept = n; putGame(s, g, gp) }),
	intPref("consoleLogCap", ScopeGame, DefaultConsoleLogCap, MinConsoleLogCap, MaxConsoleLogCap, func(s Settings, g string) int { return s.GamePrefs(g).ConsoleLogCap }, func(s *Settings, g string, n int) { gp := s.GamePrefs(g); gp.ConsoleLogCap = n; putGame(s, g, gp) }),
	strPref("nxmDefaultProfile", ScopeGame, func(s Settings, g string) string { return s.GamePrefs(g).NxmDefaultProfile }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.NxmDefaultProfile = v; putGame(s, g, gp) }),
	enumPref("cosmeticConflicts", ScopeGame, CosmeticCollapsed, cosmeticValues, func(s Settings, g string) string { return s.GamePrefs(g).CosmeticConflicts }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.CosmeticConflicts = v; putGame(s, g, gp) }),
	enumPref("enableRequirements", ScopeGame, EnableReqAlways, enableReqValues, func(s Settings, g string) string { return s.GamePrefs(g).EnableRequirements }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.EnableRequirements = v; putGame(s, g, gp) }),
	enumPref("missingRequirements", ScopeGame, MissingReqAsk, missingReqValues, func(s Settings, g string) string { return s.GamePrefs(g).MissingRequirements }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.MissingRequirements = v; putGame(s, g, gp) }),
	enumPref("smapiBuilds", ScopeGame, SmapiBuildsShow, smapiBuildsValues, func(s Settings, g string) string { return s.GamePrefs(g).SmapiBuilds }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.SmapiBuilds = v; putGame(s, g, gp) }),
	overridable(enumPref("defaultLaunchMethod", ScopeGame, LaunchSteam, launchMethodValues, func(s Settings, g string) string { return s.GamePrefs(g).DefaultLaunchMethod }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.DefaultLaunchMethod = v; putGame(s, g, gp) })),
	overridable(ptrPref("showSmapiConsole", ScopeGame, true, func(s Settings, g string) *bool { return s.GamePrefs(g).ShowSmapiConsole }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.ShowSmapiConsole = &on
		putGame(s, g, gp)
	})),
	overridable(boolPref("skipPlayCheck", ScopeGame, func(s Settings, g string) bool { return s.GamePrefs(g).SkipPlayCheck }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.SkipPlayCheck = on
		putGame(s, g, gp)
	})),
	enumPref("consoleLevel", ScopeGame, ConsoleLevelInfo, consoleLevelValues, func(s Settings, g string) string { return s.GamePrefs(g).ConsoleLevel }, func(s *Settings, g, v string) { gp := s.GamePrefs(g); gp.ConsoleLevel = v; putGame(s, g, gp) }),
	ptrPref("consoleTimestamps", ScopeGame, true, func(s Settings, g string) *bool { return s.GamePrefs(g).ConsoleTimestamps }, func(s *Settings, g string, on bool) {
		gp := s.GamePrefs(g)
		gp.ConsoleTimestamps = &on
		putGame(s, g, gp)
	}),
	ptrPref("consoleFollow", ScopeGame, true, func(s Settings, g string) *bool { return s.GamePrefs(g).ConsoleFollow }, func(s *Settings, g string, on bool) { gp := s.GamePrefs(g); gp.ConsoleFollow = &on; putGame(s, g, gp) }),
}

func defaultGameSettings() GameSettings {
	return GameSettings{
		BackupBeforePlay:            BackupBeforePlayChanged,
		LaunchBackupsKept:           DefaultLaunchBackupsKept,
		UpdateModsBeforePlayDefault: false,
		RunsKept:                    DefaultRunsKept,
		ConsoleLogCap:               DefaultConsoleLogCap,
		NxmDefaultProfile:           "",
		CosmeticConflicts:           CosmeticCollapsed,
		EnableRequirements:          EnableReqAlways,
		MissingRequirements:         MissingReqAsk,
		SmapiBuilds:                 SmapiBuildsShow,
		DefaultLaunchMethod:         LaunchSteam,
		ShowSmapiConsole:            on(),
		ConsoleLevel:                ConsoleLevelInfo,
		ConsoleTimestamps:           on(),
		ConsoleFollow:               on(),
	}
}

// GamePrefs returns stored game prefs merged with defaults.
func (s Settings) GamePrefs(gameID string) GameSettings {
	out := defaultGameSettings()
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

// PutGame stores prefs for one game id.
func PutGame(s *Settings, gameID string, g GameSettings) {
	putGame(s, gameID, g)
}

func putGame(s *Settings, gameID string, g GameSettings) {
	if s.Games == nil {
		s.Games = map[string]*GameSettings{}
	}
	cp := g
	s.Games[gameID] = &cp
}

func gameSet(s *Settings) *GameSettings {
	if s.Games == nil {
		s.Games = map[string]*GameSettings{}
	}
	if s.Games[GameStardew] == nil {
		g := s.GamePrefs(GameStardew)
		s.Games[GameStardew] = &g
	}
	return s.Games[GameStardew]
}

func mergeGame(dst *GameSettings, src GameSettings) {
	if src.BackupBeforePlay != "" {
		dst.BackupBeforePlay = src.BackupBeforePlay
	}
	if src.LaunchBackupsKept != 0 {
		dst.LaunchBackupsKept = src.LaunchBackupsKept
	}
	dst.UpdateModsBeforePlayDefault = src.UpdateModsBeforePlayDefault
	dst.SkipPlayCheck = src.SkipPlayCheck
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
	if src.SmapiBuilds != "" {
		dst.SmapiBuilds = src.SmapiBuilds
	}
	if src.DefaultLaunchMethod != "" {
		dst.DefaultLaunchMethod = src.DefaultLaunchMethod
	}
	if src.ShowSmapiConsole != nil {
		dst.ShowSmapiConsole = src.ShowSmapiConsole
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
}

func normalizeGame(g *GameSettings) {
	d := defaultGameSettings()
	if !slices.Contains(backupBeforePlayValues, g.BackupBeforePlay) {
		g.BackupBeforePlay = d.BackupBeforePlay
	}
	if g.LaunchBackupsKept < MinLaunchBackupsKept || g.LaunchBackupsKept > MaxLaunchBackupsKept {
		g.LaunchBackupsKept = d.LaunchBackupsKept
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
	if !slices.Contains(smapiBuildsValues, g.SmapiBuilds) {
		g.SmapiBuilds = d.SmapiBuilds
	}
	if !slices.Contains(launchMethodValues, g.DefaultLaunchMethod) {
		g.DefaultLaunchMethod = d.DefaultLaunchMethod
	}
	if g.ShowSmapiConsole == nil {
		g.ShowSmapiConsole = d.ShowSmapiConsole
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
}

func validateGame(g GameSettings) error {
	if !slices.Contains(backupBeforePlayValues, g.BackupBeforePlay) {
		return fmt.Errorf("backup before play must be changed, always or never, got %q", g.BackupBeforePlay)
	}
	if g.LaunchBackupsKept < MinLaunchBackupsKept || g.LaunchBackupsKept > MaxLaunchBackupsKept {
		return fmt.Errorf("launch backups kept must be %d to %d, got %d", MinLaunchBackupsKept, MaxLaunchBackupsKept, g.LaunchBackupsKept)
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
	if !slices.Contains(smapiBuildsValues, g.SmapiBuilds) {
		return fmt.Errorf("smapi builds must be never, show or include, got %q", g.SmapiBuilds)
	}
	if !slices.Contains(launchMethodValues, g.DefaultLaunchMethod) {
		return fmt.Errorf("default launch method must be steam or direct, got %q", g.DefaultLaunchMethod)
	}
	if !slices.Contains(consoleLevelValues, g.ConsoleLevel) {
		return fmt.Errorf("console level must be trace, debug, info, warn or error, got %q", g.ConsoleLevel)
	}
	return nil
}

func migrateLegacyGameFields(s *Settings) {
	legacy := GameSettings{
		BackupBeforePlay:            s.LegacyBackupBeforePlay,
		LaunchBackupsKept:           s.LegacyLaunchBackupsKept,
		UpdateModsBeforePlayDefault: s.LegacyUpdateModsBeforePlayDefault,
		RunsKept:                    s.LegacyRunsKept,
		ConsoleLogCap:               s.LegacyConsoleLogCap,
		NxmDefaultProfile:           s.LegacyNxmDefaultProfile,
		CosmeticConflicts:           s.LegacyCosmeticConflicts,
		EnableRequirements:          s.LegacyEnableRequirements,
		MissingRequirements:         s.LegacyMissingRequirements,
		SmapiBuilds:                 s.LegacySmapiBuilds,
		DefaultLaunchMethod:         s.LegacyDefaultLaunchMethod,
		ShowSmapiConsole:            s.LegacyShowSmapiConsole,
		ConsoleLevel:                s.LegacyConsoleLevel,
		ConsoleTimestamps:           s.LegacyConsoleTimestamps,
		ConsoleFollow:               s.LegacyConsoleFollow,
	}
	if !hasLegacyGame(legacy) {
		return
	}
	cur := s.GamePrefs(GameStardew)
	mergeGame(&cur, legacy)
	putGame(s, GameStardew, cur)
	clearLegacyGame(s)
}

func hasLegacyGame(g GameSettings) bool {
	return g.BackupBeforePlay != "" || g.LaunchBackupsKept != 0 || g.UpdateModsBeforePlayDefault ||
		g.RunsKept != 0 || g.ConsoleLogCap != 0 || g.NxmDefaultProfile != "" ||
		g.CosmeticConflicts != "" || g.EnableRequirements != "" || g.MissingRequirements != "" ||
		g.SmapiBuilds != "" || g.DefaultLaunchMethod != "" || g.ShowSmapiConsole != nil ||
		g.ConsoleLevel != "" || g.ConsoleTimestamps != nil || g.ConsoleFollow != nil
}

func clearLegacyGame(s *Settings) {
	s.LegacyBackupBeforePlay = ""
	s.LegacyLaunchBackupsKept = 0
	s.LegacyUpdateModsBeforePlayDefault = false
	s.LegacyRunsKept = 0
	s.LegacyConsoleLogCap = 0
	s.LegacyNxmDefaultProfile = ""
	s.LegacyCosmeticConflicts = ""
	s.LegacyEnableRequirements = ""
	s.LegacyMissingRequirements = ""
	s.LegacySmapiBuilds = ""
	s.LegacyDefaultLaunchMethod = ""
	s.LegacyShowSmapiConsole = nil
	s.LegacyConsoleLevel = ""
	s.LegacyConsoleTimestamps = nil
	s.LegacyConsoleFollow = nil
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

// AllPrefs returns every CLI-visible app-scoped setting as key/value pairs.
func (s Settings) AllPrefs() [][2]string {
	return s.AllPrefsGame("")
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

// ApplyKey writes one app-scoped CLI setting onto s.
func ApplyKey(s *Settings, key, value string) error {
	return ApplyKeyGame(s, key, value, "")
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

func enumPref(key, scope, def string, values []string, get func(Settings, string) string, set func(*Settings, string, string)) pref {
	return pref{
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeEnum, Default: def, Values: values, LabelKey: "settings." + key},
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

func intPref(key, scope string, def, lo, hi int, get func(Settings, string) int, set func(*Settings, string, int)) pref {
	return pref{
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeInt, Default: strconv.Itoa(def), Min: lo, Max: hi, LabelKey: "settings." + key},
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
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeBool, Default: "false", LabelKey: "settings." + key},
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

func ptrPref(key, scope string, def bool, get func(Settings, string) *bool, set func(*Settings, string, bool)) pref {
	return pref{
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeBool, Default: strconv.FormatBool(def), LabelKey: "settings." + key},
		get: func(s Settings, g string) string {
			v := get(s, g)
			if v == nil {
				return strconv.FormatBool(def)
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
		spec: PrefSpec{Key: key, Scope: scope, Type: TypeString, Default: "", LabelKey: "settings." + key},
		get:  get,
		set: func(s *Settings, game, raw string) error {
			set(s, game, raw)
			return nil
		},
	}
}
