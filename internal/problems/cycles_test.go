package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

func opt(id string) manifest.Dependency { return manifest.Dependency{UniqueID: id} }

func TestDependencyCycles(t *testing.T) {
	pack := inst("p", "Pack", "1.0", true)
	pack.ContentPackFor = "Frame"
	cases := []struct {
		name      string
		mods      []framework.Mod
		want      string
		blocksAll bool
	}{
		{
			"self",
			[]framework.Mod{inst("a", "Self", "1.0", true, opt("self"))},
			"Self lists itself as a dependency, so SMAPI skips it", true,
		},
		{"two via optional", []framework.Mod{
			inst("b", "Cape", "1.0", true, opt("Annetta")),
			inst("a", "Annetta", "1.0", true, req("Cape", "")),
		}, "Annetta and Cape each wait for the other, so SMAPI skips at least one of them", false},
		{"three required, through ContentPackFor", []framework.Mod{
			pack,
			inst("f", "Frame", "1.0", true, req("Lib", "")),
			inst("l", "Lib", "1.0", true, req("Pack", "")),
			inst("x", "Outside", "1.0", true, req("Lib", "")),
		}, "Frame → Lib → Pack → Frame wait for each other in a loop, so SMAPI loads none of them", true},
		{"no loop", []framework.Mod{
			inst("a", "A", "1.0", true, req("B", ""), opt("C")),
			inst("b", "B", "1.0", true, opt("C")),
			inst("c", "C", "1.0", true, opt("Off"), opt("Absent")),
			inst("o", "Off", "1.0", false, req("C", "")),
		}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enabled := make([]framework.Mod, 0, len(tc.mods))
			for _, m := range tc.mods {
				if m.Enabled {
					enabled = append(enabled, m)
				}
			}
			got := dependencyCycles(enabled)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("got %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Status != "cycle" || got[0].Summary != tc.want || got[0].CycleBlocksAll != tc.blocksAll {
				t.Fatalf("got %+v", got)
			}
			if got[0].Name != got[0].Cycle[0].Name {
				t.Fatalf("row names %s, loop starts at %s", got[0].Name, got[0].Cycle[0].Name)
			}
		})
	}
}
