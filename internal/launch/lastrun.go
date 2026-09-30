package launch

import (
	"slices"
	"strings"
)

// ModRef is an installed mod the SMAPI log column can name.
type ModRef struct {
	Name     string
	UniqueID string
}

// ModRunIssues is ERROR/ALERT and WARN counts for one installed mod
// from a completed run's SMAPI log.
type ModRunIssues struct {
	Name     string `json:"name"`
	UniqueID string `json:"uniqueId"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
}

// MatchModColumn maps a SMAPI log mod column the way the Console does:
// the whole column must equal an installed Name or UniqueID.
func MatchModColumn(column string, mods []ModRef) (ModRef, bool) {
	key := strings.TrimSpace(column)
	if key == "" {
		return ModRef{}, false
	}
	for _, m := range mods {
		if m.Name == key || m.UniqueID == key {
			return m, true
		}
	}
	return ModRef{}, false
}

// AttributeLog counts ERROR, ALERT and WARN lines per installed mod.
// Unmatched columns and continuation lines are ignored. A run with no
// matching lines for a mod yields no entry, so a newer clean run clears it.
func AttributeLog(log string, mods []ModRef) []ModRunIssues {
	byKey := map[string]*ModRunIssues{}
	for _, e := range ParseLog(log) {
		if e.Cont {
			continue
		}
		mod, ok := MatchModColumn(e.Mod, mods)
		if !ok {
			continue
		}
		key := mod.UniqueID
		if key == "" {
			key = mod.Name
		}
		row := byKey[key]
		if row == nil {
			row = &ModRunIssues{Name: mod.Name, UniqueID: mod.UniqueID}
			byKey[key] = row
		}
		switch e.Level {
		case Error, Alert:
			row.Errors++
		case Warn:
			row.Warnings++
		case Trace, Debug, Info:
		}
	}
	out := make([]ModRunIssues, 0, len(byKey))
	for _, row := range byKey {
		if row.Errors == 0 && row.Warnings == 0 {
			continue
		}
		out = append(out, *row)
	}
	slices.SortFunc(out, func(a, b ModRunIssues) int {
		if a.Errors != b.Errors {
			return b.Errors - a.Errors
		}
		if a.Warnings != b.Warnings {
			return b.Warnings - a.Warnings
		}
		return strings.Compare(a.UniqueID, b.UniqueID)
	})
	return out
}
