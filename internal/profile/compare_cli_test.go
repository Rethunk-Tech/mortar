package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestCompareProfilesCLI(t *testing.T) {
	t.Parallel()
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

func TestCompareProfilesCLINamesDifferentSources(t *testing.T) {
	t.Parallel()
	mods := []Component{{ID: "bepinex:Me.More", Name: "More", Version: "1.0.0"}}
	a := Profile{Entries: []Entry{{Key: "a", Source: Source{Kind: KindNexus, ModID: 1, FileID: 1}, Mods: mods}}}
	b := Profile{Entries: []Entry{{Key: "b", Source: Source{Kind: KindThunderstore, Name: "Me-More", Version: "1.0.0"}, Mods: mods}}}
	got := CompareProfilesCLI(a, b)
	if len(got.DifferentSource) != 1 || got.DifferentSource[0].A.Source.Kind != KindNexus || got.DifferentSource[0].B.Source.Kind != KindThunderstore {
		t.Fatalf("source differences: %#v", got.DifferentSource)
	}
	if len(got.Identical) != 0 || len(got.OnlyA) != 0 || len(got.OnlyB) != 0 || len(got.DifferentVersion) != 0 {
		t.Fatalf("the same mod from two sources is one mod: %#v", got)
	}
}
