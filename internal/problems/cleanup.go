package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// cleanupHints lists framework mods only disabled mods need, with what the frameworks found unused.
func cleanupHints(mods []Installed, found []Cleanup) []Cleanup {
	enabledNeeds := map[string]bool{}
	disabledDependents := map[string]bool{}
	frameworkIDs := map[string]bool{}
	for _, c := range found {
		frameworkIDs[c.ID.Fold()] = true
	}
	for _, im := range mods {
		for _, dep := range im.Dependencies {
			recordCleanupDependency(im, dep.ModID(), enabledNeeds, disabledDependents)
		}
		if im.ContentPackFor != "" {
			recordCleanupDependency(im, im.ContentPackForID(), enabledNeeds, disabledDependents)
		}
	}

	out := make([]Cleanup, 0)
	for _, im := range mods {
		id := im.ModID().Fold()
		if id == "" || im.ContentPackFor != "" || frameworkIDs[id] || enabledNeeds[id] || !disabledDependents[id] {
			continue
		}
		out = append(out, Cleanup{Key: im.Key, ID: im.ModID(), Name: im.Name})
	}
	out = append(out, found...)
	slices.SortFunc(out, func(a, b Cleanup) int {
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return strings.Compare(a.ID.Fold(), b.ID.Fold())
	})
	return out
}

func recordCleanupDependency(im Installed, uniqueID mod.ID, enabledNeeds, disabledDependents map[string]bool) {
	id := uniqueID.Fold()
	if id == "" {
		return
	}
	if im.Enabled {
		enabledNeeds[id] = true
	} else {
		disabledDependents[id] = true
	}
}
