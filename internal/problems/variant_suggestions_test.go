package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

func TestVariantSettingsSuggestsOnlyValueMappedToEnabledMod(t *testing.T) {
	recolourManifest := manifest.Manifest{
		Name:     "Recolour Pack",
		UniqueID: "Example.RecolourPack",
	}
	fishManifest := manifest.Manifest{
		Name:     "Fish Recolour",
		UniqueID: "Example.FishRecolour",
	}
	packMod := Installed{
		Key:      "recolour-pack",
		Manifest: recolourManifest,
	}
	pack := cachedPack{
		schema: map[string]cpSchema{
			"selection": {
				key:          "Selection",
				allowValues:  []string{"Vanilla", "Fish"},
				defaultValue: "Vanilla",
			},
		},
		patches: []cpPatch{
			{
				tokenName:  "RecolourSelection",
				tokenValue: "{{Selection}}",
				when: cpWhen{
					anyOf: [][]string{{"Example.FishRecolour"}},
					config: []cpConfig{{
						field:  "Selection",
						values: []string{"Fish"},
					}},
				},
			},
			{
				tokenName:  "RecolourSelection",
				tokenValue: "Vanilla",
				when: cpWhen{
					anyOf: [][]string{{"Example.VanillaRecolour"}},
				},
			},
		},
	}
	present := map[string]bool{"example.fishrecolour": true}
	byID := map[string]Installed{
		"example.fishrecolour": {
			Manifest: fishManifest,
		},
	}

	got := variantSettings(packMod, pack, map[string]string{"selection": "Vanilla"}, present, byID)
	if len(got) != 1 {
		t.Fatalf("variantSettings() returned %d hints, want 1: %#v", len(got), got)
	}
	if !got[0].Variant || len(got[0].Suggested) != 1 || got[0].Suggested[0] != "Fish" {
		t.Fatalf("variantSettings() = %#v, want a Fish suggestion", got[0])
	}
}
