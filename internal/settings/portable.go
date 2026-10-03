package settings

import (
	"encoding/json"
	"fmt"
	"slices"
)

const exportVersion = 1

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
	"desktopDownloadFinished", "desktopDownloadFailed", "desktopRunCrashed", "desktopModUpdates",
	"density", "theme", "gridCardSize", "showAuthorOnCards", "reduceMotion", "profileHero",
	"reuseFomodChoices", "driftChecks",
	"autoInstallMortarUpdates", "autoTrackNexus",
	"lanName", "lanAutoAcceptSameAccount", "downloadFolder",
	"profileOrder", "autoRetryDownloads", "pauseDownloadsWhilePlaying", "sidebarBadges",
	"shareIncludeDisabledMods", "shareIncludeFomodChoices", "shareIncludeNotes", "shareIncludeConfigFiles",
	"updateDigest", "verifyNexusMD5", "launchAtLogin", "startMinimised", "rememberWindow", "extensionConnection",
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
	out := make(map[string]json.RawMessage, len(portableFields)+1)
	out["version"] = json.RawMessage(strconvVersion())
	for k, v := range src {
		if _, ok := allow[k]; ok {
			out[k] = v
		}
	}
	return out
}

func strconvVersion() []byte {
	return []byte(fmt.Sprintf("%d", exportVersion))
}

// MarshalExport writes a versioned JSON of s without secrets or machine-specific fields.
func MarshalExport(s Settings) ([]byte, error) {
	raw, err := asObject(s)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(filterPortableRaw(raw), "", "  ")
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
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return Settings{}, nil, err
	}
	ver, ok := raw["version"]
	if !ok {
		return Settings{}, nil, fmt.Errorf("missing version")
	}
	var version int
	if err := json.Unmarshal(ver, &version); err != nil {
		return Settings{}, nil, err
	}
	if version != exportVersion {
		return Settings{}, nil, fmt.Errorf("unsupported settings version %d", version)
	}
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

func previewChanges(
	cur Settings,
	exported Settings,
	present map[string]struct{},
) []Change {
	next := cur
	ApplyExport(&next, exported, present)
	curRaw, _ := asObject(cur)
	nextRaw, _ := asObject(next)
	var out []Change
	for _, field := range portableFields {
		if !has(present, field) {
			continue
		}
		from, to := string(curRaw[field]), string(nextRaw[field])
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
