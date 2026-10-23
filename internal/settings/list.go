package settings

import (
	"fmt"
	"slices"
	"strings"
)

const (
	defaultListSortColumn = "name"
	defaultListSortDir    = "asc"
	defaultListGroupBy    = "status"
)

var (
	knownListColumns = []string{
		"on", "name", "version", "pluginVersion", "latest", "released", "id", "kind", "author", "source", "category",
		"endorsements", "downloads", "updated", "installed", "switchedOff", "reason", "needs", "status", "notes", "lastRun", "size", "startup", "order",
	}
	defaultListColumns = []string{"on", "name", "version", "category", "installed"}
	lockedListColumns  = []string{"on", "name"}
	listSortDirs       = []string{"asc", "desc"}
	listGroupBys       = []string{"none", "status", "category", "source", "tag", "framework", "download", "author", "group", "reason"}
)

func knownListColumn(id string) bool {
	return slices.Contains(knownListColumns, id)
}

func sanitizeListColumns(ids []string) []string {
	if len(ids) == 0 {
		return slices.Clone(defaultListColumns)
	}
	seen := map[string]bool{}
	var out []string
	known := 0
	for _, id := range ids {
		if !knownListColumn(id) || seen[id] {
			continue
		}
		known++
		seen[id] = true
		out = append(out, id)
	}
	if known == 0 {
		return slices.Clone(defaultListColumns)
	}
	for i, id := range lockedListColumns {
		if seen[id] {
			continue
		}
		out = slices.Insert(out, i, id)
		seen[id] = true
	}
	return out
}

func sanitizeListSort(column, dir string) (string, string) {
	if column == "on" || !knownListColumn(column) {
		column = defaultListSortColumn
	}
	if !slices.Contains(listSortDirs, dir) {
		dir = defaultListSortDir
	}
	return column, dir
}

func sanitizeListGroupBy(by string) string {
	if !slices.Contains(listGroupBys, by) {
		return defaultListGroupBy
	}
	return by
}

func validateList(s Settings) error {
	for _, id := range s.ListColumns {
		if !knownListColumn(id) {
			return fmt.Errorf("unknown list column %q", id)
		}
	}
	if s.ListSortColumn != "" && s.ListSortColumn != "on" && !knownListColumn(s.ListSortColumn) {
		return fmt.Errorf("unknown list sort column %q", s.ListSortColumn)
	}
	if s.ListSortDir != "" && !slices.Contains(listSortDirs, s.ListSortDir) {
		return fmt.Errorf("unknown list sort direction %q", s.ListSortDir)
	}
	if s.ListGroupBy != "" && !slices.Contains(listGroupBys, s.ListGroupBy) {
		return fmt.Errorf("unknown list group %q", s.ListGroupBy)
	}
	return nil
}

func normalizeList(s *Settings) {
	s.ListColumns = sanitizeListColumns(s.ListColumns)
	s.ListSortColumn, s.ListSortDir = sanitizeListSort(s.ListSortColumn, s.ListSortDir)
	s.ListGroupBy = sanitizeListGroupBy(s.ListGroupBy)
}

// Browse card stats, in the order a card shows them.
var browseCardStatValues = []string{"popularity", "downloads", "size", "version", "updated"}

const defaultBrowseCardStats = "popularity,downloads"

// sanitizeBrowseCardStats keeps the known ids of raw once each, in display order; "none" or no known id left over a
// non-empty raw is "none", and an empty raw is the default.
func sanitizeBrowseCardStats(raw string) string {
	if raw == "" {
		return defaultBrowseCardStats
	}
	ids := strings.Split(raw, ",")
	var out []string
	for _, id := range browseCardStatValues {
		if slices.Contains(ids, id) {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ",")
}

// browseCardStatsPref is the Browse card stats as a comma-separated list of ids, or "none".
func browseCardStatsPref() pref {
	return pref{
		spec: PrefSpec{Key: "browseCardStats", Scope: ScopeApp, Type: TypeString},
		get:  func(s Settings, _ string) string { return s.BrowseCardStats },
		set: func(s *Settings, _, raw string) error {
			for id := range strings.SplitSeq(raw, ",") {
				if id != "none" && !slices.Contains(browseCardStatValues, id) {
					return fmt.Errorf("unknown card stat %q", id)
				}
			}
			s.BrowseCardStats = sanitizeBrowseCardStats(raw)
			return nil
		},
	}
}
