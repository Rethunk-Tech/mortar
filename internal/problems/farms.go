package problems

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

// SaveFarm is one save's farm: Game1.whichFarm and, when that is 7 (a custom farm, Meadowlands included), the
// farm's Data/AdditionalFarms id from whichModFarm, "" when the save did not say.
type SaveFarm struct {
	Which int
	Mod   string
}

// whichFarmTypes are the FarmType values of Game1.whichFarm 0 to 6.
var whichFarmTypes = []string{"standard", "riverland", "forest", "hilltop", "wilderness", "fourcorners", "beach"}

var farmLabels = map[string]string{
	"standard": "Standard Farm", "riverland": "Riverland Farm", "forest": "Forest Farm", "hilltop": "Hill-top Farm",
	"wilderness": "Wilderness Farm", "fourcorners": "Four Corners Farm", "beach": "Beach Farm", "meadowlandsfarm": "Meadowlands Farm",
}

// scopeToSaveFarms marks cosmetic each conflict in which one pack's clashing patches all need a farm type none of
// the saves is; it stays listed, with the farm named. A custom farm save that does not name its farm plays every
// custom farm. With no saves nothing changes.
func scopeToSaveFarms(conflicts []framework.AssetConflict, saves []SaveFarm) []framework.AssetConflict {
	if len(saves) == 0 {
		return conflicts
	}
	have := map[string]bool{}
	anyCustom := false
	for _, s := range saves {
		switch {
		case s.Which >= 0 && s.Which < len(whichFarmTypes):
			have[whichFarmTypes[s.Which]] = true
		case s.Mod != "":
			have[strings.ToLower(s.Mod)] = true
		default:
			anyCustom = true
		}
	}
	played := func(t string) bool {
		if slices.Contains(whichFarmTypes, t) {
			return have[t]
		}
		return anyCustom || have[strings.ToLower(t)]
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
				labels[j] = cmp.Or(labels[j], t)
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
