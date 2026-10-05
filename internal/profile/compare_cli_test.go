package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestCompareProfilesCLI(t *testing.T) {
	a := Profile{Entries: []Entry{
		{Key: "a", Mods: []Component{{ID: "smapi:Alpha.Mod", Name: "Alpha", Version: "1"}}},
		{Key: "b", Mods: []Component{{ID: "smapi:Beta.Mod", Name: "Beta", Version: "1"}}},
		{Key: "c", Disabled: []mod.ID{"smapi:Gamma.Mod"}, Mods: []Component{{ID: "smapi:Gamma.Mod", Name: "Gamma", Version: "1"}}},
	}}
	b := Profile{Entries: []Entry{
		{Key: "a2", Mods: []Component{{ID: "smapi:alpha.mod", Name: "Alpha", Version: "2"}}},
		{Key: "c2", Mods: []Component{{ID: "smapi:Gamma.Mod", Name: "Gamma", Version: "1"}}},
		{Key: "d", Mods: []Component{{ID: "smapi:Delta.Mod", Name: "Delta", Version: "1"}}},
	}}

	got := CompareProfilesCLI(a, b)
	if len(got.DifferentVersion) != 1 || got.DifferentVersion[0].ID != "smapi:Alpha.Mod" {
		t.Fatalf("version differences: %#v", got.DifferentVersion)
	}
	if len(got.DifferentEnabled) != 1 || got.DifferentEnabled[0].ID != "smapi:Gamma.Mod" {
		t.Fatalf("enabled differences: %#v", got.DifferentEnabled)
	}
	if len(got.OnlyA) != 1 || got.OnlyA[0].ID != "smapi:Beta.Mod" {
		t.Fatalf("only A: %#v", got.OnlyA)
	}
	if len(got.OnlyB) != 1 || got.OnlyB[0].ID != "smapi:Delta.Mod" {
		t.Fatalf("only B: %#v", got.OnlyB)
	}
	if len(got.Identical) != 0 {
		t.Fatalf("identical: %#v", got.Identical)
	}
}
