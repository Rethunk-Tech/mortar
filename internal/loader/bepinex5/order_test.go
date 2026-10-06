package bepinex5

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestLastRunOrderHoldsForThePluginsItLoaded(t *testing.T) {
	plugin := func(guid, version string, deps ...dotnet.Relation) dotnet.Loaded {
		return dotnet.Loaded{GUID: guid, Name: guid + " Name", Version: version, Deps: deps}
	}
	order := []dotnet.Loaded{
		plugin("a", "1.0.0"),
		plugin("b", "1.1.0", dotnet.Relation{GUID: "BepInEx", Kind: dotnet.HardDependency},
			dotnet.Relation{GUID: "gone", Kind: dotnet.HardDependency}, dotnet.Relation{GUID: "elsewhere", Kind: dotnet.SoftDependency},
			dotnet.Relation{GUID: "a", Kind: dotnet.SoftDependency}),
		plugin("new", "1.0.0"),
		plugin("c", "2.0.0"),
	}
	log := "[Info   :   BepInEx] Loading [c Name 2.0]\r\n" +
		"[Info   :   BepInEx] Loading [b Name 1.1.0]\r\n" +
		"[Info   :   BepInEx] Loading [a Name 1.0.0]\r\n" +
		"[Info   :   BepInEx] Loading [Mortar BepInEx Bridge 0.1.0]\r\n"
	rows := rows(order, asLastRun(order, log))
	var got []string
	var marked []string
	for _, r := range rows {
		got = append(got, string(r.ID))
		if r.LastRun {
			marked = append(marked, string(r.ID))
		}
	}
	id := func(g string) mod.ID { return mod.NewID(mod.FormatBepInEx, g) }
	if want := []string{"bepinex:c", "bepinex:b", "bepinex:new", "bepinex:a"}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
	if want := []string{"bepinex:c", "bepinex:a"}; !slices.Equal(marked, want) {
		t.Fatalf("marked %v, want %v", marked, want)
	}
	b := rows[1]
	if !slices.Equal(b.Required, []mod.ID{id("gone")}) || !slices.Equal(b.MissingRequired, []mod.ID{id("gone")}) ||
		!slices.Equal(b.Optional, []mod.ID{id("a")}) || !slices.Equal(rows[3].Dependents, []mod.ID{id("b")}) {
		t.Fatalf("b %+v, a %+v", b, rows[3])
	}
}
