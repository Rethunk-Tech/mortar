package profile

import "testing"

func TestCompareProfilesCLI(t *testing.T) {
	a := Profile{Entries: []Entry{
		{Key: "a", Mods: []EntryMod{{UniqueID: "Alpha.Mod", Name: "Alpha", Version: "1"}}},
		{Key: "b", Mods: []EntryMod{{UniqueID: "Beta.Mod", Name: "Beta", Version: "1"}}},
		{Key: "c", Disabled: []string{"Gamma.Mod"}, Mods: []EntryMod{{UniqueID: "Gamma.Mod", Name: "Gamma", Version: "1"}}},
	}}
	b := Profile{Entries: []Entry{
		{Key: "a2", Mods: []EntryMod{{UniqueID: "alpha.mod", Name: "Alpha", Version: "2"}}},
		{Key: "c2", Mods: []EntryMod{{UniqueID: "Gamma.Mod", Name: "Gamma", Version: "1"}}},
		{Key: "d", Mods: []EntryMod{{UniqueID: "Delta.Mod", Name: "Delta", Version: "1"}}},
	}}

	got := CompareProfilesCLI(a, b)
	if len(got.DifferentVersion) != 1 || got.DifferentVersion[0].UniqueID != "Alpha.Mod" {
		t.Fatalf("version differences: %#v", got.DifferentVersion)
	}
	if len(got.DifferentEnabled) != 1 || got.DifferentEnabled[0].UniqueID != "Gamma.Mod" {
		t.Fatalf("enabled differences: %#v", got.DifferentEnabled)
	}
	if len(got.OnlyA) != 1 || got.OnlyA[0].UniqueID != "Beta.Mod" {
		t.Fatalf("only A: %#v", got.OnlyA)
	}
	if len(got.OnlyB) != 1 || got.OnlyB[0].UniqueID != "Delta.Mod" {
		t.Fatalf("only B: %#v", got.OnlyB)
	}
	if len(got.Identical) != 0 {
		t.Fatalf("identical: %#v", got.Identical)
	}
}
