package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func TestScopeToSaveFarms(t *testing.T) {
	conflicts := []framework.AssetConflict{
		{Target: "maps/farm_foraging", Farms: [][]string{{"forest"}, nil}},
		{Target: "maps/farm_frontier", Farms: [][]string{{"Author.Frontier_Frontier"}, {"Author.Frontier_Frontier"}}},
		{Target: "maps/farm_grandpa", Farms: [][]string{{"Grandpa.Farm"}, nil}},
		{Target: "data/objects"},
	}
	got := scopeToSaveFarms(conflicts, []SaveFarm{{Which: 0}, {Which: 7, Mod: "grandpa.farm"}})
	if !got[0].Cosmetic || got[0].Note == nil || *got[0].Note != (framework.ConflictNote{Kind: "farm", Value: "Forest Farm"}) {
		t.Fatalf("no Forest save, got %#v", got[0])
	}
	if !got[1].Cosmetic || got[1].Note.Value != "Author.Frontier_Frontier" {
		t.Fatalf("no save on the Frontier farm, got %#v", got[1])
	}
	if got[2].Cosmetic || got[3].Cosmetic {
		t.Fatalf("a save plays Grandpa's farm, got %#v", got)
	}
	if conflicts[0].Cosmetic {
		t.Fatal("the cached conflicts were changed")
	}
	if got := scopeToSaveFarms(conflicts, []SaveFarm{{Which: 2}, {Which: 7}}); got[0].Cosmetic || got[1].Cosmetic || got[2].Cosmetic {
		t.Fatalf("a Forest save and a custom one that does not name its farm play all of these, got %#v", got)
	}
	if got := scopeToSaveFarms(conflicts, nil); got[0].Cosmetic {
		t.Fatalf("without saves nothing is scoped, got %#v", got)
	}
}
