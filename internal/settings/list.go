package settings

import (
	"fmt"
	"slices"
)

const (
	defaultListSortColumn = "name"
	defaultListSortDir    = "asc"
	defaultListGroupBy    = "status"
)

var (
	knownListColumns = []string{
		"on", "name", "version", "latest", "uniqueId", "author", "source", "category",
		"endorsements", "downloads", "updated", "installed", "needs", "status", "notes", "lastRun",
	}
	defaultListColumns = []string{"on", "name", "version", "author", "source", "category", "status"}
	lockedListColumns  = []string{"on", "name"}
	listSortDirs       = []string{"asc", "desc"}
	listGroupBys       = []string{"none", "status", "category", "source", "tag", "framework", "author", "group"}
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
