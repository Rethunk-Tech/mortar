package problems

import (
	"slices"
	"strings"
)

func cleanupHints(mods []Installed) []Cleanup {
	enabledNeeds := map[string]bool{}
	disabledDependents := map[string]bool{}
	for _, mod := range mods {
		for _, dep := range mod.Dependencies {
			recordCleanupDependency(mod, dep.UniqueID, enabledNeeds, disabledDependents)
		}
		if mod.ContentPackFor != "" {
			recordCleanupDependency(mod, mod.ContentPackFor, enabledNeeds, disabledDependents)
		}
	}

	out := make([]Cleanup, 0)
	for _, mod := range mods {
		id := strings.ToLower(strings.TrimSpace(mod.UniqueID))
		if id == "" || mod.ContentPackFor != "" || enabledNeeds[id] || !disabledDependents[id] {
			continue
		}
		out = append(out, Cleanup{Key: mod.Key, UniqueID: mod.UniqueID, Name: mod.Name})
	}
	slices.SortFunc(out, func(a, b Cleanup) int {
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.UniqueID), strings.ToLower(b.UniqueID))
	})
	return out
}

func recordCleanupDependency(mod Installed, uniqueID string, enabledNeeds, disabledDependents map[string]bool) {
	id := strings.ToLower(strings.TrimSpace(uniqueID))
	if id == "" {
		return
	}
	if mod.Enabled {
		enabledNeeds[id] = true
	} else {
		disabledDependents[id] = true
	}
}
