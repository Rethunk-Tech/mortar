package contentpatcher

import (
	"testing"

	cp "github.com/Rethunk-Tech/go-contentpatcher"
)

func TestVariantSettingsSuggestsOnlyValueMappedToEnabledMod(t *testing.T) {
	packMod := cp.Mod{Key: "recolour-pack", Name: "Recolour Pack", UniqueID: cp.SMAPI("Example.RecolourPack")}
	pack := cp.Pack{
		Schema: map[string]cp.Schema{
			"selection": {
				Key:          "Selection",
				AllowValues:  []string{"Vanilla", "Fish"},
				DefaultValue: "Vanilla",
			},
		},
		Patches: []cp.Patch{
			{
				TokenName:  "RecolourSelection",
				TokenValue: "{{Selection}}",
				When: cp.When{
					AnyOf: [][]string{{"example.fishrecolour"}},
					Config: []cp.Config{{
						Field:  "Selection",
						Values: []string{"Fish"},
					}},
				},
			},
			{
				TokenName:  "RecolourSelection",
				TokenValue: "Vanilla",
				When: cp.When{
					AnyOf: [][]string{{"example.vanillarecolour"}},
				},
			},
		},
	}
	present := map[string]bool{"example.fishrecolour": true}
	byID := map[string]cp.Mod{
		"example.fishrecolour": {UniqueID: cp.SMAPI("Example.FishRecolour"), Name: "Fish Recolour"},
	}

	got := variantSettings(packMod, pack, map[string]string{"selection": "Vanilla"}, present, byID)
	if len(got) != 1 {
		t.Fatalf("variantSettings() returned %d hints, want 1: %#v", len(got), got)
	}
	if !got[0].Variant || len(got[0].Suggested) != 1 || got[0].Suggested[0] != "Fish" {
		t.Fatalf("variantSettings() = %#v, want a Fish suggestion", got[0])
	}
}
