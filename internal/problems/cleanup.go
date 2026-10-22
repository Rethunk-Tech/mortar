package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// orphanReason is why an implicit mod no switched-on mod requires or can use is offered for removal.
const orphanReason = "Brought in for other mods, and none that is switched on needs it"

// cleanupHints lists framework mods only disabled mods need, implicit mods nothing switched on needs (a content pack
// is left to its framework's own check, since maps use tilesheet packs without naming them), and what the frameworks
// found unused. A mod switched-on content names in its conditions is in use too.
func cleanupHints(mods []framework.Mod, found []framework.Cleanup, mentioned []string) []framework.Cleanup {
	enabledNeeds := map[string]bool{}
	for _, id := range mentioned {
		enabledNeeds[id] = true
	}
	disabledDependents := map[string]bool{}
	frameworkIDs := map[string]bool{}
	for _, c := range found {
		frameworkIDs[c.ID.Fold()] = true
	}
	for _, im := range mods {
		for _, dep := range im.Dependencies {
			// A switched-on mod keeps what it can use as well as what it requires: removing an optional dependency
			// takes away the feature it adds.
			if !dep.Required && !im.Enabled {
				continue
			}
			recordCleanupDependency(im, dep.ModID(), enabledNeeds, disabledDependents)
		}
		if im.ContentPackFor != "" {
			recordCleanupDependency(im, im.ContentPackForID(), enabledNeeds, disabledDependents)
		}
	}

	out := make([]framework.Cleanup, 0)
	for _, im := range mods {
		id := im.ModID().Fold()
		if id == "" || im.ContentPackFor != "" || frameworkIDs[id] || enabledNeeds[id] || !disabledDependents[id] {
			continue
		}
		out = append(out, framework.Cleanup{Key: im.Key, ID: im.ModID(), Name: im.Name})
	}
	for _, im := range mods {
		id := im.ModID().Fold()
		if !im.Implicit || id == "" || im.ContentPackFor != "" || im.SourceKind == "smapi" || im.SourceKind == "mortar" ||
			frameworkIDs[id] || enabledNeeds[id] || disabledDependents[id] {
			continue
		}
		out = append(out, framework.Cleanup{Key: im.Key, ID: im.ModID(), Name: im.Name, Reason: orphanReason, Code: "orphan"})
	}
	out = append(out, found...)
	slices.SortFunc(out, func(a, b framework.Cleanup) int {
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return strings.Compare(a.ID.Fold(), b.ID.Fold())
	})
	return out
}

func recordCleanupDependency(im framework.Mod, uniqueID mod.ID, enabledNeeds, disabledDependents map[string]bool) {
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
