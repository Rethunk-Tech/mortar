package settings

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const exportVersion = 1

// Portable is settings.json minus secrets and machine-specific or personal fields.
type Portable struct {
	Version                      int                      `json:"version"`
	Language                     string                   `json:"language"`
	Accent                       string                   `json:"accent"`
	Background                   string                   `json:"background"`
	LastGame                     string                   `json:"lastGame"`
	BackupsKept                  int                      `json:"backupsKept"`
	ListColumns                  []string                 `json:"listColumns"`
	ListSortColumn               string                   `json:"listSortColumn"`
	ListSortDir                  string                   `json:"listSortDir"`
	ListGroupBy                  string                   `json:"listGroupBy"`
	CheckModUpdatesOnStart       *bool                    `json:"checkModUpdatesOnStart"`
	TellWhenSmapiOut             *bool                    `json:"tellWhenSmapiOut"`
	KeepInTray                   bool                     `json:"keepInTray"`
	IncludeBetaReleases          bool                     `json:"includeBetaReleases"`
	IncludePrereleaseModVersions bool                     `json:"includePrereleaseModVersions"`
	CheckOnlyEnabledMods         bool                     `json:"checkOnlyEnabledMods"`
	EnableModsWhenInstalled      *bool                    `json:"enableModsWhenInstalled"`
	TipsSeen                     []string                 `json:"tipsSeen"`
	NexusPreferredDownloadServer string                   `json:"nexusPreferredDownloadServer"`
	NxmRedirectOtherGames        *bool                    `json:"nxmRedirectOtherGames"`
	OnPlay                       string                   `json:"onPlay"`
	ParallelDownloads            int                      `json:"parallelDownloads"`
	UpdateCheckIntervalMinutes   int                      `json:"updateCheckIntervalMinutes"`
	NotifyModUpdates             *bool                    `json:"notifyModUpdates"`
	KeepDownloadArchives         bool                     `json:"keepDownloadArchives"`
	StoreRetentionDays           int                      `json:"storeRetentionDays"`
	DefaultModsView              string                   `json:"defaultModsView"`
	ConfirmRemovals              *bool                    `json:"confirmRemovals"`
	BackgroundBadgeChecks        *bool                    `json:"backgroundBadgeChecks"`
	StartScreen                  string                   `json:"startScreen"`
	Dates                        string                   `json:"dates"`
	TrashRetentionDays           int                      `json:"trashRetentionDays"`
	HistoryEventsKept            int                      `json:"historyEventsKept"`
	NotifyDownloadFinished       *bool                    `json:"notifyDownloadFinished"`
	NotifyDownloadFailed         *bool                    `json:"notifyDownloadFailed"`
	NotifyRunCrashed             *bool                    `json:"notifyRunCrashed"`
	Density                      string                   `json:"density"`
	GridCardSize                 string                   `json:"gridCardSize"`
	ShowAuthorOnCards            *bool                    `json:"showAuthorOnCards"`
	ReduceMotion                 string                   `json:"reduceMotion"`
	ProfileHero                  string                   `json:"profileHero"`
	ReuseFomodChoices            *bool                    `json:"reuseFomodChoices"`
	DriftChecks                  *bool                    `json:"driftChecks"`
	AutoInstallMortarUpdates     *bool                    `json:"autoInstallMortarUpdates"`
	AutoTrackNexus               bool                     `json:"autoTrackNexus"`
	LanName                      string                   `json:"lanName"`
	LanAutoAcceptSameAccount     bool                     `json:"lanAutoAcceptSameAccount"`
	DownloadFolder               string                   `json:"downloadFolder"`
	ProfileOrder                 string                   `json:"profileOrder"`
	AutoRetryDownloads           string                   `json:"autoRetryDownloads"`
	PauseDownloadsWhilePlaying   bool                     `json:"pauseDownloadsWhilePlaying"`
	SidebarBadges                string                   `json:"sidebarBadges"`
	ShareIncludeDisabledMods     *bool                    `json:"shareIncludeDisabledMods"`
	ShareIncludeFomodChoices     *bool                    `json:"shareIncludeFomodChoices"`
	ShareIncludeNotes            *bool                    `json:"shareIncludeNotes"`
	ShareIncludeConfigFiles      *bool                    `json:"shareIncludeConfigFiles"`
	VerifyNexusMD5               bool                     `json:"verifyNexusMD5"`
	LaunchAtLogin                bool                     `json:"launchAtLogin"`
	StartMinimised               bool                     `json:"startMinimised"`
	RememberWindow               bool                     `json:"rememberWindow"`
	ExtensionConnection          string                   `json:"extensionConnection"`
	Games                        map[string]*GameSettings `json:"games"`

	// Legacy root game fields, accepted on import then moved into Games.
	BackupBeforePlay            string `json:"backupBeforePlay,omitempty"`
	LaunchBackupsKept           int    `json:"launchBackupsKept,omitempty"`
	UpdateModsBeforePlayDefault bool   `json:"updateModsBeforePlayDefault,omitempty"`
	RunsKept                    int    `json:"runsKept,omitempty"`
	ConsoleLogCap               int    `json:"consoleLogCap,omitempty"`
	NxmDefaultProfile           string `json:"nxmDefaultProfile,omitempty"`
	CosmeticConflicts           string `json:"cosmeticConflicts,omitempty"`
	EnableRequirements          string `json:"enableRequirements,omitempty"`
	MissingRequirements         string `json:"missingRequirements,omitempty"`
	SmapiBuilds                 string `json:"smapiBuilds,omitempty"`
	DefaultLaunchMethod         string `json:"defaultLaunchMethod,omitempty"`
	ShowSmapiConsole            *bool  `json:"showSmapiConsole,omitempty"`
	ConsoleLevel                string `json:"consoleLevel,omitempty"`
	ConsoleTimestamps           *bool  `json:"consoleTimestamps,omitempty"`
	ConsoleFollow               *bool  `json:"consoleFollow,omitempty"`
}

// Change is one field that import would replace.
type Change struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// ImportPreview is a validated export ready to apply, or empty Raw when the dialog was cancelled.
type ImportPreview struct {
	Raw     string   `json:"raw"`
	Changes []Change `json:"changes"`
}

var portableFields = []string{
	"language", "accent", "background", "lastGame", "backupsKept",
	"listColumns", "listSortColumn", "listSortDir", "listGroupBy",
	"checkModUpdatesOnStart", "tellWhenSmapiOut", "keepInTray", "includeBetaReleases",
	"includePrereleaseModVersions", "checkOnlyEnabledMods", "enableModsWhenInstalled", "tipsSeen",
	"nexusPreferredDownloadServer", "nxmRedirectOtherGames",
	"onPlay", "parallelDownloads", "updateCheckIntervalMinutes",
	"notifyModUpdates", "keepDownloadArchives", "storeRetentionDays",
	"defaultModsView", "confirmRemovals", "backgroundBadgeChecks",
	"startScreen", "dates", "trashRetentionDays", "historyEventsKept",
	"notifyDownloadFinished", "notifyDownloadFailed", "notifyRunCrashed",
	"density", "gridCardSize", "showAuthorOnCards", "reduceMotion", "profileHero",
	"reuseFomodChoices", "driftChecks",
	"autoInstallMortarUpdates", "autoTrackNexus",
	"lanName", "lanAutoAcceptSameAccount", "downloadFolder",
	"profileOrder", "autoRetryDownloads", "pauseDownloadsWhilePlaying", "sidebarBadges",
	"shareIncludeDisabledMods", "shareIncludeFomodChoices", "shareIncludeNotes", "shareIncludeConfigFiles",
	"verifyNexusMD5", "launchAtLogin", "startMinimised", "rememberWindow", "extensionConnection",
	"games",
}

func has(present map[string]struct{}, key string) bool {
	_, ok := present[key]
	return ok
}

// MarshalExport writes a versioned JSON of s without secrets or machine-specific fields.
func MarshalExport(s Settings) ([]byte, error) {
	return json.MarshalIndent(fillPortable(s), "", "  ")
}

// ParseExport validates an export. Unknown fields are ignored; each portable value is sanitised as on load.
func ParseExport(b []byte) (Portable, map[string]struct{}, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return Portable{}, nil, err
	}
	if _, ok := raw["version"]; !ok {
		return Portable{}, nil, fmt.Errorf("missing version")
	}
	var p Portable
	if err := json.Unmarshal(b, &p); err != nil {
		return Portable{}, nil, err
	}
	if p.Version != exportVersion {
		return Portable{}, nil, fmt.Errorf("unsupported settings version %d", p.Version)
	}
	present := make(map[string]struct{}, len(raw))
	for k := range raw {
		present[k] = struct{}{}
	}
	overlay := Defaults()
	copyPortable(&overlay, p, present)
	overlay = sanitizePortable(overlay)
	out := fillPortable(overlay)
	out.Version = exportVersion
	return out, present, nil
}

// ApplyExport copies sanitised portable fields that were present in the file onto cur.
func ApplyExport(cur *Settings, p Portable, present map[string]struct{}) {
	copyPortable(cur, p, present)
}

func sanitizePortable(s Settings) Settings {
	if !slices.Contains(accents, s.Accent) {
		s.Accent = Defaults().Accent
	}
	if !slices.Contains(backgrounds, s.Background) {
		s.Background = Defaults().Background
	}
	if s.BackupsKept < MinBackupsKept || s.BackupsKept > MaxBackupsKept {
		s.BackupsKept = Defaults().BackupsKept
	}
	normalizeToggles(&s)
	normalizePrefs(&s)
	normalizeList(&s)
	normalizeTips(&s)
	normalizeNexus(&s)
	return s
}

func previewChanges(cur Settings, p Portable, present map[string]struct{}) []Change {
	next := cur
	ApplyExport(&next, p, present)
	var out []Change
	for _, field := range portableFields {
		if field == "games" {
			if !has(present, "games") && !legacyGamePresent(present) {
				continue
			}
		} else if !has(present, field) {
			continue
		}
		from, to := fieldText(cur, field), fieldText(next, field)
		if from == to {
			continue
		}
		out = append(out, Change{Field: field, From: from, To: to})
	}
	if out == nil {
		out = []Change{}
	}
	return out
}

func legacyGamePresent(present map[string]struct{}) bool {
	for _, k := range []string{
		"backupBeforePlay", "launchBackupsKept", "updateModsBeforePlayDefault",
		"runsKept", "consoleLogCap", "nxmDefaultProfile", "cosmeticConflicts",
		"enableRequirements", "missingRequirements", "smapiBuilds",
		"defaultLaunchMethod", "showSmapiConsole", "consoleLevel",
		"consoleTimestamps", "consoleFollow",
	} {
		if has(present, k) {
			return true
		}
	}
	return false
}

func fieldText(s Settings, field string) string {
	if field == "games" {
		b, err := json.Marshal(s.Games)
		if err != nil {
			return ""
		}
		return string(b)
	}
	if v, err := s.LookupGame(field, GameStardew); err == nil {
		return v
	}
	switch field {
	case "language":
		return s.Language
	case "accent":
		return s.Accent
	case "background":
		return s.Background
	case "lastGame":
		return s.LastGame
	case "backupsKept":
		return strconv.Itoa(s.BackupsKept)
	case "listColumns":
		return strings.Join(s.ListColumns, ", ")
	case "tellWhenSmapiOut":
		return boolText(s.TellWhenSmapiOut)
	case "keepInTray":
		return strconv.FormatBool(s.KeepInTray)
	case "includeBetaReleases":
		return strconv.FormatBool(s.IncludeBetaReleases)
	case "includePrereleaseModVersions":
		return strconv.FormatBool(s.IncludePrereleaseModVersions)
	case "checkOnlyEnabledMods":
		return strconv.FormatBool(s.CheckOnlyEnabledMods)
	case "enableModsWhenInstalled":
		return boolText(s.EnableModsWhenInstalled)
	case "tipsSeen":
		return strings.Join(s.TipsSeen, ", ")
	case "nexusPreferredDownloadServer":
		if s.NexusPreferredDownloadServer == "" {
			return "automatic"
		}
		return s.NexusPreferredDownloadServer
	case "nxmRedirectOtherGames":
		return strconv.FormatBool(s.RedirectOtherGames())
	default:
		return ""
	}
}

func boolText(v *bool) string {
	if v == nil {
		return ""
	}
	return strconv.FormatBool(*v)
}

func fillPortable(s Settings) Portable {
	games := map[string]*GameSettings{}
	for id, g := range s.Games {
		if g == nil {
			continue
		}
		cp := *g
		normalizeGame(&cp)
		games[id] = &cp
	}
	return Portable{
		Version:                      exportVersion,
		Language:                     s.Language,
		Accent:                       s.Accent,
		Background:                   s.Background,
		LastGame:                     s.LastGame,
		BackupsKept:                  s.BackupsKept,
		ListColumns:                  slices.Clone(s.ListColumns),
		ListSortColumn:               s.ListSortColumn,
		ListSortDir:                  s.ListSortDir,
		ListGroupBy:                  s.ListGroupBy,
		CheckModUpdatesOnStart:       s.CheckModUpdatesOnStart,
		TellWhenSmapiOut:             s.TellWhenSmapiOut,
		KeepInTray:                   s.KeepInTray,
		IncludeBetaReleases:          s.IncludeBetaReleases,
		IncludePrereleaseModVersions: s.IncludePrereleaseModVersions,
		CheckOnlyEnabledMods:         s.CheckOnlyEnabledMods,
		EnableModsWhenInstalled:      s.EnableModsWhenInstalled,
		TipsSeen:                     slices.Clone(s.TipsSeen),
		NexusPreferredDownloadServer: s.NexusPreferredDownloadServer,
		NxmRedirectOtherGames:        s.NxmRedirectOtherGames,
		OnPlay:                       s.OnPlay,
		ParallelDownloads:            s.ParallelDownloads,
		UpdateCheckIntervalMinutes:   s.UpdateCheckIntervalMinutes,
		NotifyModUpdates:             s.NotifyModUpdates,
		KeepDownloadArchives:         s.KeepDownloadArchives,
		StoreRetentionDays:           s.StoreRetentionDays,
		DefaultModsView:              s.DefaultModsView,
		ConfirmRemovals:              s.ConfirmRemovals,
		BackgroundBadgeChecks:        s.BackgroundBadgeChecks,
		StartScreen:                  s.StartScreen,
		Dates:                        s.Dates,
		TrashRetentionDays:           s.TrashRetentionDays,
		HistoryEventsKept:            s.HistoryEventsKept,
		NotifyDownloadFinished:       s.NotifyDownloadFinished,
		NotifyDownloadFailed:         s.NotifyDownloadFailed,
		NotifyRunCrashed:             s.NotifyRunCrashed,
		Density:                      s.Density,
		GridCardSize:                 s.GridCardSize,
		ShowAuthorOnCards:            s.ShowAuthorOnCards,
		ReduceMotion:                 s.ReduceMotion,
		ProfileHero:                  s.ProfileHero,
		ReuseFomodChoices:            s.ReuseFomodChoices,
		DriftChecks:                  s.DriftChecks,
		AutoInstallMortarUpdates:     s.AutoInstallMortarUpdates,
		AutoTrackNexus:               s.AutoTrackNexus,
		LanName:                      s.LanName,
		LanAutoAcceptSameAccount:     s.LanAutoAcceptSameAccount,
		DownloadFolder:               s.DownloadFolder,
		ProfileOrder:                 s.ProfileOrder,
		AutoRetryDownloads:           s.AutoRetryDownloads,
		PauseDownloadsWhilePlaying:   s.PauseDownloadsWhilePlaying,
		SidebarBadges:                s.SidebarBadges,
		ShareIncludeDisabledMods:     s.ShareIncludeDisabledMods,
		ShareIncludeFomodChoices:     s.ShareIncludeFomodChoices,
		ShareIncludeNotes:            s.ShareIncludeNotes,
		ShareIncludeConfigFiles:      s.ShareIncludeConfigFiles,
		VerifyNexusMD5:               s.VerifyNexusMD5,
		LaunchAtLogin:                s.LaunchAtLogin,
		StartMinimised:               s.StartMinimised,
		RememberWindow:               s.RememberWindow,
		ExtensionConnection:          s.ExtensionConnection,
		Games:                        games,
	}
}

func copyPortable(dst *Settings, p Portable, present map[string]struct{}) {
	if has(present, "language") {
		dst.Language = p.Language
	}
	if has(present, "accent") {
		dst.Accent = p.Accent
	}
	if has(present, "background") {
		dst.Background = p.Background
	}
	if has(present, "lastGame") {
		dst.LastGame = p.LastGame
	}
	if has(present, "backupsKept") {
		dst.BackupsKept = p.BackupsKept
	}
	if has(present, "listColumns") {
		dst.ListColumns = slices.Clone(p.ListColumns)
	}
	if has(present, "listSortColumn") {
		dst.ListSortColumn = p.ListSortColumn
	}
	if has(present, "listSortDir") {
		dst.ListSortDir = p.ListSortDir
	}
	if has(present, "listGroupBy") {
		dst.ListGroupBy = p.ListGroupBy
	}
	if has(present, "checkModUpdatesOnStart") {
		dst.CheckModUpdatesOnStart = p.CheckModUpdatesOnStart
	}
	if has(present, "tellWhenSmapiOut") {
		dst.TellWhenSmapiOut = p.TellWhenSmapiOut
	}
	if has(present, "keepInTray") {
		dst.KeepInTray = p.KeepInTray
	}
	if has(present, "includeBetaReleases") {
		dst.IncludeBetaReleases = p.IncludeBetaReleases
	}
	if has(present, "includePrereleaseModVersions") {
		dst.IncludePrereleaseModVersions = p.IncludePrereleaseModVersions
	}
	if has(present, "checkOnlyEnabledMods") {
		dst.CheckOnlyEnabledMods = p.CheckOnlyEnabledMods
	}
	if has(present, "enableModsWhenInstalled") {
		dst.EnableModsWhenInstalled = p.EnableModsWhenInstalled
	}
	if has(present, "tipsSeen") {
		dst.TipsSeen = slices.Clone(p.TipsSeen)
	}
	if has(present, "nexusPreferredDownloadServer") {
		dst.NexusPreferredDownloadServer = p.NexusPreferredDownloadServer
	}
	if has(present, "nxmRedirectOtherGames") {
		dst.NxmRedirectOtherGames = p.NxmRedirectOtherGames
	}
	if has(present, "onPlay") {
		dst.OnPlay = p.OnPlay
	}
	if has(present, "parallelDownloads") {
		dst.ParallelDownloads = p.ParallelDownloads
	}
	if has(present, "updateCheckIntervalMinutes") {
		dst.UpdateCheckIntervalMinutes = p.UpdateCheckIntervalMinutes
	}
	if has(present, "notifyModUpdates") {
		dst.NotifyModUpdates = p.NotifyModUpdates
	}
	if has(present, "keepDownloadArchives") {
		dst.KeepDownloadArchives = p.KeepDownloadArchives
	}
	if has(present, "storeRetentionDays") {
		dst.StoreRetentionDays = p.StoreRetentionDays
	}
	if has(present, "defaultModsView") {
		dst.DefaultModsView = p.DefaultModsView
	}
	if has(present, "confirmRemovals") {
		dst.ConfirmRemovals = p.ConfirmRemovals
	}
	if has(present, "backgroundBadgeChecks") {
		dst.BackgroundBadgeChecks = p.BackgroundBadgeChecks
	}
	if has(present, "startScreen") {
		dst.StartScreen = p.StartScreen
	}
	if has(present, "dates") {
		dst.Dates = p.Dates
	}
	if has(present, "trashRetentionDays") {
		dst.TrashRetentionDays = p.TrashRetentionDays
	}
	if has(present, "historyEventsKept") {
		dst.HistoryEventsKept = p.HistoryEventsKept
	}
	if has(present, "notifyDownloadFinished") {
		dst.NotifyDownloadFinished = p.NotifyDownloadFinished
	}
	if has(present, "notifyDownloadFailed") {
		dst.NotifyDownloadFailed = p.NotifyDownloadFailed
	}
	if has(present, "notifyRunCrashed") {
		dst.NotifyRunCrashed = p.NotifyRunCrashed
	}
	if has(present, "density") {
		dst.Density = p.Density
	}
	if has(present, "gridCardSize") {
		dst.GridCardSize = p.GridCardSize
	}
	if has(present, "showAuthorOnCards") {
		dst.ShowAuthorOnCards = p.ShowAuthorOnCards
	}
	if has(present, "reduceMotion") {
		dst.ReduceMotion = p.ReduceMotion
	}
	if has(present, "profileHero") {
		dst.ProfileHero = p.ProfileHero
	}
	if has(present, "reuseFomodChoices") {
		dst.ReuseFomodChoices = p.ReuseFomodChoices
	}
	if has(present, "driftChecks") {
		dst.DriftChecks = p.DriftChecks
	}
	if has(present, "autoInstallMortarUpdates") {
		dst.AutoInstallMortarUpdates = p.AutoInstallMortarUpdates
	}
	if has(present, "autoTrackNexus") {
		dst.AutoTrackNexus = p.AutoTrackNexus
	}
	if has(present, "lanName") {
		dst.LanName = p.LanName
	}
	if has(present, "lanAutoAcceptSameAccount") {
		dst.LanAutoAcceptSameAccount = p.LanAutoAcceptSameAccount
	}
	if has(present, "downloadFolder") {
		dst.DownloadFolder = p.DownloadFolder
	}
	if has(present, "profileOrder") {
		dst.ProfileOrder = p.ProfileOrder
	}
	if has(present, "autoRetryDownloads") {
		dst.AutoRetryDownloads = p.AutoRetryDownloads
	}
	if has(present, "pauseDownloadsWhilePlaying") {
		dst.PauseDownloadsWhilePlaying = p.PauseDownloadsWhilePlaying
	}
	if has(present, "sidebarBadges") {
		dst.SidebarBadges = p.SidebarBadges
	}
	if has(present, "shareIncludeDisabledMods") {
		dst.ShareIncludeDisabledMods = p.ShareIncludeDisabledMods
	}
	if has(present, "shareIncludeFomodChoices") {
		dst.ShareIncludeFomodChoices = p.ShareIncludeFomodChoices
	}
	if has(present, "shareIncludeNotes") {
		dst.ShareIncludeNotes = p.ShareIncludeNotes
	}
	if has(present, "shareIncludeConfigFiles") {
		dst.ShareIncludeConfigFiles = p.ShareIncludeConfigFiles
	}
	if has(present, "verifyNexusMD5") {
		dst.VerifyNexusMD5 = p.VerifyNexusMD5
	}
	if has(present, "launchAtLogin") {
		dst.LaunchAtLogin = p.LaunchAtLogin
	}
	if has(present, "startMinimised") {
		dst.StartMinimised = p.StartMinimised
	}
	if has(present, "rememberWindow") {
		dst.RememberWindow = p.RememberWindow
	}
	if has(present, "extensionConnection") {
		dst.ExtensionConnection = p.ExtensionConnection
	}
	if has(present, "games") && p.Games != nil {
		if dst.Games == nil {
			dst.Games = map[string]*GameSettings{}
		}
		for id, g := range p.Games {
			if g == nil {
				continue
			}
			cp := *g
			dst.Games[id] = &cp
		}
	}
	legacy := GameSettings{
		BackupBeforePlay:            p.BackupBeforePlay,
		LaunchBackupsKept:           p.LaunchBackupsKept,
		UpdateModsBeforePlayDefault: p.UpdateModsBeforePlayDefault,
		RunsKept:                    p.RunsKept,
		ConsoleLogCap:               p.ConsoleLogCap,
		NxmDefaultProfile:           p.NxmDefaultProfile,
		CosmeticConflicts:           p.CosmeticConflicts,
		EnableRequirements:          p.EnableRequirements,
		MissingRequirements:         p.MissingRequirements,
		SmapiBuilds:                 p.SmapiBuilds,
		DefaultLaunchMethod:         p.DefaultLaunchMethod,
		ShowSmapiConsole:            p.ShowSmapiConsole,
		ConsoleLevel:                p.ConsoleLevel,
		ConsoleTimestamps:           p.ConsoleTimestamps,
		ConsoleFollow:               p.ConsoleFollow,
	}
	if hasLegacyGame(legacy) && legacyGamePresent(present) {
		cur := dst.GamePrefs(GameStardew)
		mergeGame(&cur, legacy)
		putGame(dst, GameStardew, cur)
	}
}
