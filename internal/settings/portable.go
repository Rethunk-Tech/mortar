package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

const exportVersion = 1

var portableFields = []string{
	"language", "accent", "font", "background", "lastGame",
	"listColumns", "listSortColumn", "listSortDir", "listGroupBy",
	"checkModUpdatesOnStart", "askEndorseMods", "smapiBuilds", "loaderPrefs", "showSmapiConsole", "keepInTray", "includeBetaReleases",
	"includePrereleaseModVersions", "checkOnlyEnabledMods", "enableModsWhenInstalled", "tipsSeen",
	"nexusPreferredDownloadServer", "nxmRedirectOtherGames",
	"onPlay", "parallelDownloads", "downloadSpeedLimit", "updateCheckIntervalMinutes",
	"notifyModUpdates", "keepDownloadArchives", "storeRetentionDays",
	"defaultModsView", "implicitModsView", "confirmRemovals", "backgroundBadgeChecks",
	"startScreen", "antivirus", "dates", "graphics", "trashRetentionDays", "historyEventsKept",
	"notifyDownloadFinished", "notifyDownloadFailed", "notifyRunCrashed",
	"desktopDownloadFinished", "desktopDownloadFailed", "desktopRunCrashed", "desktopModUpdates",
	"density", "uiScale", "theme", "gridCardSize", "showAuthorOnCards", "markLocalMods", "centerConfigControls", "browseCardStats", "reduceMotion", "touchRipples", "profileHero",
	"reuseFomodChoices", "driftChecks",
	"autoInstallMortarUpdates", "autoTrackNexus",
	"lanSharing", "lanPort", "lanName", "lanAutoAcceptPaired",
	"profileOrder", "autoRetryDownloads", "thunderstoreHost", "sidebarBadges",
	"shareIncludeDisabledMods", "shareIncludeFomodChoices", "shareIncludeNotes", "shareIncludeConfigFiles", "shareIncludeProblemChoices", "shareIncludeLibraries",
	"updateDigest", "verifyNexusMD5", "showAdultContent", "launchAtLogin", "startMinimised", "rememberWindow", "systemTitleBar", "extensionConnection",
	"games",
}

func has(present map[string]struct{}, key string) bool {
	_, ok := present[key]
	return ok
}

func portableSet() map[string]struct{} {
	out := make(map[string]struct{}, len(portableFields))
	for _, k := range portableFields {
		out[k] = struct{}{}
	}
	return out
}

func asObject(v any) (map[string]json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func filterPortableRaw(src map[string]json.RawMessage) map[string]json.RawMessage {
	allow := portableSet()
	out := make(map[string]json.RawMessage, len(portableFields))
	for k, v := range src {
		if _, ok := allow[k]; ok {
			out[k] = v
		}
	}
	return out
}

// exportFile is an export: the version, then the portable settings by scope, as in settings.json.
type exportFile struct {
	Version int `json:"version"`
	scoped
}

// MarshalExport writes a versioned JSON of s without secrets or machine-specific fields.
func MarshalExport(s Settings) ([]byte, error) {
	// What an import would drop anyway stays out of the file: the player's folders, commands and local progress.
	s.Games = maps.Clone(s.Games)
	for id, g := range s.Games {
		if g != nil {
			c := *g
			c.LaunchPresetTemplates = slices.Clone(g.LaunchPresetTemplates)
			s.Games[id] = &c
		}
	}
	keepLocalOnly(&s, Settings{}, true)
	raw, err := asObject(s)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(exportFile{Version: exportVersion, scoped: split(filterPortableRaw(raw))}, "", "  ")
}

func overlayPortable(cur Settings, raw map[string]json.RawMessage, present map[string]struct{}) (Settings, error) {
	dst, err := asObject(cur)
	if err != nil {
		return Settings{}, err
	}
	allow := portableSet()
	for k, v := range raw {
		if !has(present, k) {
			continue
		}
		if _, ok := allow[k]; !ok {
			continue
		}
		dst[k] = v
	}
	b, err := json.Marshal(dst)
	if err != nil {
		return Settings{}, err
	}
	var out Settings
	if err := json.Unmarshal(b, &out); err != nil {
		return Settings{}, err
	}
	out = sanitizePortable(out)
	if avRank(out.Antivirus) < avRank(cur.Antivirus) {
		out.Antivirus = cur.Antivirus
	}
	keepLocalOnly(&out, cur, has(present, "games"))
	return out, nil
}

// avRank orders the antivirus modes by strictness: an imported file may raise the mode, never lower it.
func avRank(mode string) int {
	switch mode {
	case AntivirusOff:
		return 0
	case AntivirusAutomatic:
		return 1
	}
	return 2
}

// keepLocalOnly puts back, from cur, the per-game fields that name a program, a build or a place Mortar reads mods
// from or writes to. An imported file never sets them, so what a file carries there is dropped and the player's own
// stay. A launch preset keeps its name and options but loses its prefix command and environment.
func keepLocalOnly(out *Settings, cur Settings, gamesImported bool) {
	// A loader's pin travels; where this machine's sweep and toast stand does not.
	prefs := map[string]LoaderPrefs{}
	for k, p := range out.LoaderPrefs {
		p.LastSweepVersion, p.ToastAt = "", ""
		prefs[k] = p
	}
	for k, c := range cur.LoaderPrefs {
		p := prefs[k]
		p.LastSweepVersion, p.ToastAt = c.LastSweepVersion, c.ToastAt
		prefs[k] = p
	}
	maps.DeleteFunc(prefs, func(_ string, p LoaderPrefs) bool { return p == LoaderPrefs{} })
	out.LoaderPrefs = nil
	if len(prefs) > 0 {
		out.LoaderPrefs = prefs
	}
	fromFile := map[string]bool{}
	if gamesImported {
		for id := range out.Games {
			fromFile[id] = true
		}
	}
	for id, g := range cur.Games {
		if g != nil && out.Games[id] == nil {
			if out.Games == nil {
				out.Games = map[string]*GameSettings{}
			}
			c := *g
			c.LaunchPresetTemplates = slices.Clone(g.LaunchPresetTemplates)
			out.Games[id] = &c
		}
	}
	for id, dst := range out.Games {
		if dst == nil {
			continue
		}
		var local GameSettings
		if g := cur.Games[id]; g != nil {
			local = *g
		}
		dst.DevProjects = local.DevProjects
		dst.ExtraModsFolder = local.ExtraModsFolder
		dst.ExtraModsProfiles = local.ExtraModsProfiles
		dst.SyncOwnFolder = local.SyncOwnFolder
		dst.BackupLocation = local.BackupLocation
		dst.LastDownloadsSeen = local.LastDownloadsSeen
		dst.LastSweepGameVersion = local.LastSweepGameVersion
		dst.NxmDefaultProfile = local.NxmDefaultProfile
		if !fromFile[id] {
			continue
		}
		for i := range dst.LaunchPresetTemplates {
			dst.LaunchPresetTemplates[i].Prefix = ""
			dst.LaunchPresetTemplates[i].Env = ""
		}
	}
}

// ParseExport validates an export. Unknown fields are ignored; each portable value is sanitised as on load.
func ParseExport(b []byte) (Settings, map[string]struct{}, error) {
	var file struct {
		Version *int `json:"version"`
		scoped
	}
	if err := json.Unmarshal(b, &file); err != nil {
		return Settings{}, nil, err
	}
	if file.Version == nil {
		return Settings{}, nil, fmt.Errorf("missing version")
	}
	if *file.Version != exportVersion {
		return Settings{}, nil, fmt.Errorf("unsupported settings version %d", *file.Version)
	}
	raw := file.flatten()
	raw["version"] = json.RawMessage(fmt.Sprint(exportVersion))
	present := make(map[string]struct{}, len(raw))
	for k := range raw {
		present[k] = struct{}{}
	}
	out, err := overlayPortable(Defaults(), raw, present)
	if err != nil {
		return Settings{}, nil, err
	}
	return out, present, nil
}

// ApplyExport copies sanitised portable fields that were present in the file onto cur.
func ApplyExport(cur *Settings, exported Settings, present map[string]struct{}) {
	// The scanner's command and socket never travel, so a mode that needs one is no use without this machine's own.
	if exported.Antivirus == AntivirusCommand && cur.AntivirusCommand == "" || exported.Antivirus == AntivirusClamd && cur.AntivirusSocket == "" {
		exported.Antivirus = cur.Antivirus
	}
	raw, err := asObject(exported)
	if err != nil {
		return
	}
	for _, k := range portableFields {
		if has(present, k) {
			if _, ok := raw[k]; !ok {
				raw[k] = json.RawMessage("null")
			}
		}
	}
	next, err := overlayPortable(*cur, raw, present)
	if err != nil {
		return
	}
	*cur = next
}

func sanitizePortable(s Settings) Settings {
	if !validAccent(s.Accent) {
		s.Accent = Defaults().Accent
	}
	if !slices.Contains(backgrounds, s.Background) {
		s.Background = Defaults().Background
	}
	normalizeToggles(&s)
	normalizePrefs(&s)
	normalizeList(&s)
	normalizeTips(&s)
	normalizeNexus(&s)
	return s
}
