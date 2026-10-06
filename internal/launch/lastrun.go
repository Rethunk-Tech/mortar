package launch

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// ModRef is an installed mod the SMAPI log column can name.
type ModRef struct {
	Name          string `json:"name"`
	ID            mod.ID `json:"id"`
	Key           string `json:"key"`
	Version       string `json:"version"`
	SourceVersion string `json:"sourceVersion"`
}

// ModRunIssues is ERROR/ALERT and WARN counts for one installed mod
// from a completed run's SMAPI log.
type ModRunIssues struct {
	Name     string `json:"name"`
	ID       mod.ID `json:"id"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
}

// MatchModColumn maps a SMAPI log mod column the way the Console does:
// the whole column must equal an installed Name or SMAPI unique id.
func MatchModColumn(column string, mods []ModRef) (ModRef, bool) {
	key := strings.TrimSpace(column)
	if key == "" {
		return ModRef{}, false
	}
	for _, m := range mods {
		if m.Name == key || m.ID.Local() == key {
			return m, true
		}
	}
	return ModRef{}, false
}

// AttributeLog counts ERROR, ALERT and WARN lines per installed mod. A column that names no mod is looked up, lower-cased,
// in aliases: the GUIDs and names of a BepInEx package's plugins. Unmatched columns and continuation lines are ignored. A run with no
// matching lines for a mod yields no entry, so a newer clean run clears it.
func AttributeLog(log string, mods []ModRef, aliases map[string]ModRef) []ModRunIssues {
	byKey := map[string]*ModRunIssues{}
	for _, e := range ParseLog(log) {
		if e.Cont {
			continue
		}
		ref, ok := MatchModColumn(e.Mod, mods)
		if !ok {
			ref, ok = aliases[strings.ToLower(strings.TrimSpace(e.Mod))]
		}
		if !ok {
			continue
		}
		key := string(ref.ID)
		if key == "" {
			key = ref.Name
		}
		row := byKey[key]
		if row == nil {
			row = &ModRunIssues{Name: ref.Name, ID: ref.ID}
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
		return strings.Compare(string(a.ID), string(b.ID))
	})
	return out
}
