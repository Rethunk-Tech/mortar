package dotnet

import (
	"slices"
	"testing"
)

func TestLoadOrderWalksGUIDsCaselessAndPutsDependenciesFirstInDeclaredOrder(t *testing.T) {
	asm := func(guid, version string, deps ...string) Declared {
		d := Declared{Plugins: []Plugin{{GUID: guid, Name: guid, Version: version}}}
		for _, g := range deps {
			d.Relations = append(d.Relations, Relation{Plugin: guid, GUID: g, Kind: SoftDependency})
		}
		return d
	}
	order := LoadOrder([]Declared{
		asm("b", "1.0.0", "Z", "y"),
		asm("y", "1.0.0"),
		asm("Z", "1.0.0"),
		// An older copy's dependency does not count: the newer copy is the one that loads.
		asm("a", "1.0.0", "b"),
		asm("a", "2.0.0"),
		asm("c", "1.0.0", "d"),
		asm("d", "1.0.0", "c"),
	})
	var got []string
	for _, l := range order {
		got = append(got, l.GUID)
		if l.Cycle != (l.GUID == "c" || l.GUID == "d") {
			t.Errorf("%s cycle %v", l.GUID, l.Cycle)
		}
	}
	if want := []string{"a", "Z", "y", "b", "d", "c"}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
}
