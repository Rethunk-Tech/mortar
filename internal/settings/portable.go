package settings

import (
	"encoding/json"
	"fmt"
	"slices"
)

const exportVersion = 1

var portableFields = []string{
	"language", "accent", "background", "lastGame",
	"listColumns", "listSortColumn", "listSortDir", "listGroupBy",
	"checkModUpdatesOnStart", "tellWhenSmapiOut", "smapiBuilds", "smapiPin", "showSmapiConsole", "keepInTray", "includeBetaReleases",
	"includePrereleaseModVersions", "checkOnlyEnabledMods", "enableModsWhenInstalled", "tipsSeen",
	"nexusPreferredDownloadServer", "nxmRedirectOtherGames",
	"onPlay", "parallelDownloads", "updateCheckIntervalMinutes",
	"notifyModUpdates", "keepDownloadArchives", "storeRetentionDays",
	"defaultModsView", "confirmRemovals", "backgroundBadgeChecks",
	"startScreen", "dates", "trashRetentionDays", "historyEventsKept",
	"notifyDownloadFinished", "notifyDownloadFailed", "notifyRunCrashed",
	"desktopDownloadFinished", "desktopDownloadFailed", "desktopRunCrashed", "desktopModUpdates",
	"density", "theme", "gridCardSize", "showAuthorOnCards", "reduceMotion", "profileHero",
	"reuseFomodChoices", "driftChecks",
	"autoInstallMortarUpdates", "autoTrackNexus",
	"lanName", "lanAutoAcceptSameAccount", "downloadFolder", "watchFolders",
	"profileOrder", "autoRetryDownloads", "pauseDownloadsWhilePlaying", "sidebarBadges",
	"shareIncludeDisabledMods", "shareIncludeFomodChoices", "shareIncludeNotes", "shareIncludeConfigFiles",
	"updateDigest", "verifyNexusMD5", "showAdultContent", "launchAtLogin", "startMinimised", "rememberWindow", "extensionConnection",
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
	return sanitizePortable(out), nil
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
	if !slices.Contains(accents, s.Accent) {
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
