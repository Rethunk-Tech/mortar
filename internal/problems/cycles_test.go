package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
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

func TestPluginCyclesFollowBepInExsSort(t *testing.T) {
	pkg := func(key string, enabled bool) profile.PackageRef {
		return profile.PackageRef{Key: key, ID: mod.ID("thunderstore:Ns-" + key), Name: key, Enabled: enabled}
	}
	plugin := func(guid, name string, rel ...dotnet.Relation) dotnet.Declared {
		for i := range rel {
			rel[i].Plugin = guid
		}
		return dotnet.Declared{Plugins: []dotnet.Plugin{{GUID: guid, Name: name, Version: "1.0.0"}}, Relations: rel}
	}
	hard := func(guid string) dotnet.Relation { return dotnet.Relation{GUID: guid, Kind: dotnet.HardDependency} }
	soft := func(guid string) dotnet.Relation { return dotnet.Relation{GUID: guid, Kind: dotnet.SoftDependency} }
	cases := []struct {
		name     string
		pkgs     []profile.PackageRef
		declared map[string]dotnet.Declared
		want     string
	}{
		{"soft links loop too, GUIDs in any case", []profile.PackageRef{pkg("b", true), pkg("a", true)}, map[string]dotnet.Declared{
			"a": plugin("me.a", "Alpha", soft("ME.B")),
			"b": plugin("me.b", "Beta", hard("me.a")),
		}, "Alpha and Beta each depend on the other, so BepInEx stops before loading any plugin"},
		{"self", []profile.PackageRef{pkg("a", true)}, map[string]dotnet.Declared{
			"a": plugin("me.a", "Alpha", hard("me.a")),
		}, "Alpha lists itself as a dependency, so BepInEx stops before loading any plugin"},
		{"a disabled package breaks the loop", []profile.PackageRef{pkg("a", true), pkg("b", false)}, map[string]dotnet.Declared{
			"a": plugin("me.a", "Alpha", hard("me.b")),
			"b": plugin("me.b", "Beta", hard("me.a")),
		}, ""},
		{"an incompatible plugin is dropped before the sort", []profile.PackageRef{pkg("a", true), pkg("b", true), pkg("c", true)}, map[string]dotnet.Declared{
			"a": plugin("me.a", "Alpha", hard("me.b")),
			"b": plugin("me.b", "Beta", hard("me.a"), dotnet.Relation{GUID: "me.c", Kind: dotnet.Incompatible}),
			"c": plugin("me.c", "Gamma"),
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pluginCycles(tc.pkgs, tc.declared)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("got %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Summary != tc.want || !got[0].CycleBlocksAll || !got[0].CycleStopsLoader || got[0].Key != got[0].Cycle[0].Key {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
