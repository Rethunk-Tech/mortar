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
	Version                      int      `json:"version"`
	Language                     string   `json:"language"`
	Accent                       string   `json:"accent"`
	Background                   string   `json:"background"`
	LastGame                     string   `json:"lastGame"`
	BackupsKept                  int      `json:"backupsKept"`
	ListColumns                  []string `json:"listColumns"`
	ListSortColumn               string   `json:"listSortColumn"`
	ListSortDir                  string   `json:"listSortDir"`
	ListGroupBy                  string   `json:"listGroupBy"`
	CheckModUpdatesOnStart       *bool    `json:"checkModUpdatesOnStart"`
	TellWhenSmapiOut             *bool    `json:"tellWhenSmapiOut"`
	KeepInTray                   bool     `json:"keepInTray"`
	IncludeBetaReleases          bool     `json:"includeBetaReleases"`
	IncludePrereleaseModVersions bool     `json:"includePrereleaseModVersions"`
	CheckOnlyEnabledMods         bool     `json:"checkOnlyEnabledMods"`
	EnableModsWhenInstalled      *bool    `json:"enableModsWhenInstalled"`
	TipsSeen                     []string `json:"tipsSeen"`
	NexusPreferredDownloadServer string   `json:"nexusPreferredDownloadServer"`
	NxmRedirectOtherGames        *bool    `json:"nxmRedirectOtherGames"`
	OnPlay                       string   `json:"onPlay"`
	BackupBeforePlay             string   `json:"backupBeforePlay"`
	LaunchBackupsKept            int      `json:"launchBackupsKept"`
	UpdateModsBeforePlayDefault  bool     `json:"updateModsBeforePlayDefault"`
	RunsKept                     int      `json:"runsKept"`
	ConsoleLogCap                int      `json:"consoleLogCap"`
	ParallelDownloads            int      `json:"parallelDownloads"`
	UpdateCheckIntervalMinutes   int      `json:"updateCheckIntervalMinutes"`
	NotifyModUpdates             *bool    `json:"notifyModUpdates"`
	KeepDownloadArchives         bool     `json:"keepDownloadArchives"`
	StoreRetentionDays           int      `json:"storeRetentionDays"`
	NxmDefaultProfile            string   `json:"nxmDefaultProfile"`
	DefaultModsView              string   `json:"defaultModsView"`
	ConfirmRemovals              *bool    `json:"confirmRemovals"`
	CosmeticConflicts            string   `json:"cosmeticConflicts"`
	BackgroundBadgeChecks        *bool    `json:"backgroundBadgeChecks"`
	StartScreen                  string   `json:"startScreen"`
	Dates                        string   `json:"dates"`
	TrashRetentionDays           int      `json:"trashRetentionDays"`
	HistoryEventsKept            int      `json:"historyEventsKept"`
	NotifyDownloadFinished       *bool    `json:"notifyDownloadFinished"`
	NotifyDownloadFailed         *bool    `json:"notifyDownloadFailed"`
	NotifyRunCrashed             *bool    `json:"notifyRunCrashed"`
	Density                      string   `json:"density"`
	GridCardSize                 string   `json:"gridCardSize"`
	ShowAuthorOnCards            *bool    `json:"showAuthorOnCards"`
	ReduceMotion                 string   `json:"reduceMotion"`
	ProfileHero                  string   `json:"profileHero"`
	EnableRequirements           string   `json:"enableRequirements"`
	MissingRequirements          string   `json:"missingRequirements"`
	ReuseFomodChoices            *bool    `json:"reuseFomodChoices"`
	DriftChecks                  *bool    `json:"driftChecks"`
	SmapiBuilds                  string   `json:"smapiBuilds"`
	AutoInstallMortarUpdates     *bool    `json:"autoInstallMortarUpdates"`
	AutoTrackNexus               bool     `json:"autoTrackNexus"`
	DefaultLaunchMethod          string   `json:"defaultLaunchMethod"`
	ShowSmapiConsole             *bool    `json:"showSmapiConsole"`
	ConsoleLevel                 string   `json:"consoleLevel"`
	ConsoleTimestamps            *bool    `json:"consoleTimestamps"`
	ConsoleFollow                *bool    `json:"consoleFollow"`
	LanName                      string   `json:"lanName"`
	LanAutoAcceptSameAccount     bool     `json:"lanAutoAcceptSameAccount"`
	DownloadFolder               string   `json:"downloadFolder"`
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
	"onPlay", "backupBeforePlay", "launchBackupsKept", "updateModsBeforePlayDefault",
	"runsKept", "consoleLogCap", "parallelDownloads", "updateCheckIntervalMinutes",
	"notifyModUpdates", "keepDownloadArchives", "storeRetentionDays", "nxmDefaultProfile",
	"defaultModsView", "confirmRemovals", "cosmeticConflicts", "backgroundBadgeChecks",
	"startScreen", "dates", "trashRetentionDays", "historyEventsKept",
	"notifyDownloadFinished", "notifyDownloadFailed", "notifyRunCrashed",
	"density", "gridCardSize", "showAuthorOnCards", "reduceMotion", "profileHero",
	"enableRequirements", "missingRequirements", "reuseFomodChoices", "driftChecks",
	"smapiBuilds", "autoInstallMortarUpdates", "autoTrackNexus", "defaultLaunchMethod",
	"showSmapiConsole", "consoleLevel", "consoleTimestamps", "consoleFollow",
	"lanName", "lanAutoAcceptSameAccount", "downloadFolder",
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
		if _, ok := present[field]; !ok {
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

func fieldText(s Settings, field string) string {
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
	case "listSortColumn":
		return s.ListSortColumn
	case "listSortDir":
		return s.ListSortDir
	case "listGroupBy":
		return s.ListGroupBy
	case "checkModUpdatesOnStart":
		return boolText(s.CheckModUpdatesOnStart)
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
	case "onPlay":
		return s.OnPlay
	case "backupBeforePlay":
		return s.BackupBeforePlay
	case "launchBackupsKept":
		return strconv.Itoa(s.LaunchBackupsKept)
	case "updateModsBeforePlayDefault":
		return strconv.FormatBool(s.UpdateModsBeforePlayDefault)
	case "runsKept":
		return strconv.Itoa(s.RunsKept)
	case "consoleLogCap":
		return strconv.Itoa(s.ConsoleLogCap)
	case "parallelDownloads":
		return strconv.Itoa(s.ParallelDownloads)
	case "updateCheckIntervalMinutes":
		return strconv.Itoa(s.UpdateCheckIntervalMinutes)
	case "notifyModUpdates":
		return boolText(s.NotifyModUpdates)
	case "keepDownloadArchives":
		return strconv.FormatBool(s.KeepDownloadArchives)
	case "storeRetentionDays":
		return strconv.Itoa(s.StoreRetentionDays)
	case "nxmDefaultProfile":
		return s.NxmDefaultProfile
	case "defaultModsView":
		return s.DefaultModsView
	case "confirmRemovals":
		return boolText(s.ConfirmRemovals)
	case "cosmeticConflicts":
		return s.CosmeticConflicts
	case "backgroundBadgeChecks":
		return boolText(s.BackgroundBadgeChecks)
	case "startScreen":
		return s.StartScreen
	case "dates":
		return s.Dates
	case "trashRetentionDays":
		return strconv.Itoa(s.TrashRetentionDays)
	case "historyEventsKept":
		return strconv.Itoa(s.HistoryEventsKept)
	case "notifyDownloadFinished":
		return boolText(s.NotifyDownloadFinished)
	case "notifyDownloadFailed":
		return boolText(s.NotifyDownloadFailed)
	case "notifyRunCrashed":
		return boolText(s.NotifyRunCrashed)
	case "density":
		return s.Density
	case "gridCardSize":
		return s.GridCardSize
	case "showAuthorOnCards":
		return boolText(s.ShowAuthorOnCards)
	case "reduceMotion":
		return s.ReduceMotion
	case "profileHero":
		return s.ProfileHero
	case "enableRequirements":
		return s.EnableRequirements
	case "missingRequirements":
		return s.MissingRequirements
	case "reuseFomodChoices":
		return boolText(s.ReuseFomodChoices)
	case "driftChecks":
		return boolText(s.DriftChecks)
	case "smapiBuilds":
		return s.SmapiBuilds
	case "autoInstallMortarUpdates":
		return boolText(s.AutoInstallMortarUpdates)
	case "autoTrackNexus":
		return strconv.FormatBool(s.AutoTrackNexus)
	case "defaultLaunchMethod":
		return s.DefaultLaunchMethod
	case "showSmapiConsole":
		return boolText(s.ShowSmapiConsole)
	case "consoleLevel":
		return s.ConsoleLevel
	case "consoleTimestamps":
		return boolText(s.ConsoleTimestamps)
	case "consoleFollow":
		return boolText(s.ConsoleFollow)
	case "lanName":
		return s.LanName
	case "lanAutoAcceptSameAccount":
		return strconv.FormatBool(s.LanAutoAcceptSameAccount)
	case "downloadFolder":
		return s.DownloadFolder
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
