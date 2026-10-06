package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func TestScopeToSaveFarms(t *testing.T) {
	conflicts := []framework.AssetConflict{
		{Target: "maps/farm_foraging", Farms: [][]string{{"forest"}, nil}},
		{Target: "maps/farm_frontier", Farms: [][]string{{"Frontier"}, {"Frontier"}}},
		{Target: "data/objects"},
	}
	got := scopeToSaveFarms(conflicts, []int{0})
	if !got[0].Cosmetic || got[0].Note == nil || *got[0].Note != (framework.ConflictNote{Kind: "farm", Value: "Forest Farm"}) {
		t.Fatalf("no Forest save, got %#v", got[0])
	}
	if !got[1].Cosmetic || got[1].Note.Value != "Frontier" || got[2].Cosmetic {
		t.Fatalf("no custom save, got %#v", got)
	}
	if conflicts[0].Cosmetic {
		t.Fatal("the cached conflicts were changed")
	}
	if got := scopeToSaveFarms(conflicts, []int{2, 7}); got[0].Cosmetic || got[1].Cosmetic {
		t.Fatalf("a Forest and a custom save play both, got %#v", got)
	}
	if got := scopeToSaveFarms(conflicts, nil); got[0].Cosmetic {
		t.Fatalf("without saves nothing is scoped, got %#v", got)
	}
}
