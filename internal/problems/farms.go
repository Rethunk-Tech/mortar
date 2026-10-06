package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

// whichFarmTypes are the FarmType values of Game1.whichFarm 0 to 6; 7 is any custom farm, Meadowlands included,
// and a save does not say which.
var whichFarmTypes = []string{"standard", "riverland", "forest", "hilltop", "wilderness", "fourcorners", "beach"}

var farmLabels = map[string]string{
	"standard": "Standard Farm", "riverland": "Riverland Farm", "forest": "Forest Farm", "hilltop": "Hill-top Farm",
	"wilderness": "Wilderness Farm", "fourcorners": "Four Corners Farm", "beach": "Beach Farm", "meadowlandsfarm": "Meadowlands Farm",
}

// scopeToSaveFarms marks cosmetic each conflict in which one pack's clashing patches all need a farm type none of
// the saves (their whichFarm values) is; it stays listed, with the farm named. With no saves nothing changes.
func scopeToSaveFarms(conflicts []framework.AssetConflict, which []int) []framework.AssetConflict {
	if len(which) == 0 {
		return conflicts
	}
	have := map[string]bool{}
	custom := false
	for _, w := range which {
		if w >= 0 && w < len(whichFarmTypes) {
			have[whichFarmTypes[w]] = true
		} else {
			custom = true
		}
	}
	played := func(t string) bool {
		if slices.Contains(whichFarmTypes, t) {
			return have[t]
		}
		return custom
	}
	var out []framework.AssetConflict
	for i, c := range conflicts {
		if c.Cosmetic {
			continue
		}
		for _, need := range c.Farms {
			if len(need) == 0 || slices.ContainsFunc(need, played) {
				continue
			}
			if out == nil {
				out = slices.Clone(conflicts)
			}
			labels := make([]string, len(need))
			for j, t := range need {
				labels[j] = farmLabels[t]
				if labels[j] == "" {
					labels[j] = t
				}
			}
			out[i].Cosmetic = true
			out[i].Note = &framework.ConflictNote{Kind: "farm", Value: strings.Join(labels, " or ")}
			break
		}
	}
	if out == nil {
		return conflicts
	}
	return out
}
