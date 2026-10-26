package settings

import (
	"encoding/json"
	"fmt"
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
	"lanSharing", "lanPort", "lanAllowAnyAddress", "lanName", "lanAutoAcceptPaired", "downloadFolder", "watchFolders", "syncFolder", "vortexFolder", "packFolder",
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
	keepLocalOnly(&out, cur)
	return out, nil
}

// keepLocalOnly puts back, from cur, the per-game fields that name a program, a build or a place Mortar reads mods
// from or writes to. An imported file never sets them, so what a file carries there is dropped and the player's own
// stay. A launch preset keeps its name and options but loses its prefix command and environment.
func keepLocalOnly(out *Settings, cur Settings) {
	for id, g := range cur.Games {
		if g != nil && out.Games[id] == nil {
			if out.Games == nil {
				out.Games = map[string]*GameSettings{}
			}
			c := *g
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
