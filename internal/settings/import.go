package settings

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode"
)

// Import section ids, in the order the import dialog lists them.
const (
	SectionGeneral       = "general"
	SectionAppearance    = "appearance"
	SectionNotifications = "notifications"
	SectionDownloads     = "downloads"
	SectionStorage       = "storage"
	SectionSharing       = "sharing"
	SectionGames         = "games"
)

var importSections = []string{
	SectionGeneral, SectionAppearance, SectionNotifications, SectionDownloads, SectionStorage, SectionSharing, SectionGames,
}

var sectionFields = map[string][]string{
	SectionAppearance: {
		"accent", "background", "theme", "density", "gridCardSize", "showAuthorOnCards", "reduceMotion", "profileHero",
		"listColumns", "listSortColumn", "listSortDir", "listGroupBy", "defaultModsView", "profileOrder", "sidebarBadges",
	},
	SectionNotifications: {
		"notifyModUpdates", "notifyDownloadFinished", "notifyDownloadFailed", "notifyRunCrashed",
		"desktopDownloadFinished", "desktopDownloadFailed", "desktopRunCrashed", "desktopModUpdates", "updateDigest",
	},
	SectionDownloads: {
		"parallelDownloads", "autoRetryDownloads", "pauseDownloadsWhilePlaying", "nexusPreferredDownloadServer",
		"nxmRedirectOtherGames", "verifyNexusMD5", "autoTrackNexus", "checkModUpdatesOnStart", "updateCheckIntervalMinutes",
		"includePrereleaseModVersions", "checkOnlyEnabledMods", "enableModsWhenInstalled", "includeBetaReleases", "showAdultContent",
		"autoInstallMortarUpdates", "backgroundBadgeChecks", "tellWhenSmapiOut", "reuseFomodChoices", "driftChecks",
	},
	SectionStorage: {"keepDownloadArchives", "storeRetentionDays", "trashRetentionDays", "historyEventsKept", "downloadFolder", "watchFolders", "syncFolder"},
	SectionSharing: {
		"shareIncludeDisabledMods", "shareIncludeFomodChoices", "shareIncludeNotes", "shareIncludeConfigFiles",
		"lanSharing", "lanName", "lanAutoAcceptPaired",
	},
	SectionGames: {"games", "smapiBuilds", "loaderPrefs", "showSmapiConsole"},
}

// ImportSections lists every section id.
func ImportSections() []string { return slices.Clone(importSections) }

func sectionOf(key string) string {
	for sec, keys := range sectionFields {
		if slices.Contains(keys, key) {
			return sec
		}
	}
	return SectionGeneral
}

// ImportChange is one setting an import would replace; From and To are display text.
type ImportChange struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// ImportSection groups the changes of one settings area.
type ImportSection struct {
	Section string         `json:"section"`
	Changes []ImportChange `json:"changes"`
}

// ImportPreview is what importing a file would change, and the fields the file holds that Mortar does not import.
type ImportPreview struct {
	Sections []ImportSection `json:"sections"`
	Ignored  []string        `json:"ignored"`
}

// PreviewImport validates the export at path and lists, per section, the settings it would change.
func PreviewImport(cur Settings, raw []byte) (ImportPreview, error) {
	p, present, err := ParseExport(raw)
	if err != nil {
		return ImportPreview{}, err
	}
	next := cur
	ApplyExport(&next, p, present)
	curRaw, _ := asObject(cur)
	nextRaw, _ := asObject(next)
	bySection := map[string][]ImportChange{}
	for _, key := range portableFields {
		if !has(present, key) {
			continue
		}
		from, to := string(curRaw[key]), string(nextRaw[key])
		if from == to {
			continue
		}
		sec := sectionOf(key)
		bySection[sec] = append(bySection[sec], ImportChange{Key: key, Label: humanize(key), From: display(from), To: display(to)})
	}
	out := ImportPreview{Sections: []ImportSection{}, Ignored: []string{}}
	for _, sec := range importSections {
		if ch := bySection[sec]; len(ch) > 0 {
			out.Sections = append(out.Sections, ImportSection{Section: sec, Changes: ch})
		}
	}
	allow := portableSet()
	for k := range present {
		if _, ok := allow[k]; !ok && k != "version" {
			out.Ignored = append(out.Ignored, k)
		}
	}
	slices.Sort(out.Ignored)
	return out, nil
}

// ApplyImport applies only the fields of the chosen sections to cur.
func ApplyImport(cur *Settings, raw []byte, sections []string) error {
	p, present, err := ParseExport(raw)
	if err != nil {
		return err
	}
	for k := range present {
		if !slices.Contains(sections, sectionOf(k)) {
			delete(present, k)
		}
	}
	ApplyExport(cur, p, present)
	return nil
}

func display(v string) string {
	var s string
	if json.Unmarshal([]byte(v), &s) == nil {
		return s
	}
	return v
}

// humanize turns a camelCase key into a fallback label; the GUI uses its own copy where it has one.
func humanize(key string) string {
	var b strings.Builder
	for i, r := range key {
		switch {
		case i == 0:
			b.WriteRune(unicode.ToUpper(r))
		case unicode.IsUpper(r):
			b.WriteByte(' ')
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
