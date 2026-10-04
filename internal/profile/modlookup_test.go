package profile

import "testing"

func TestEntryEnabledAndFindMod(t *testing.T) {
	p := Profile{Entries: []Entry{
		{Key: "a", Mods: []EntryMod{{UniqueID: "Au.One"}, {UniqueID: ""}}, Disabled: []string{" au.one "}},
		{Key: "b", Mods: []EntryMod{{UniqueID: "Au.Two"}}},
	}}
	a := p.Entries[0]
	if a.Enabled("AU.ONE") {
		t.Error("a disabled ID matches regardless of case and stray spaces")
	}
	if !a.Enabled("") {
		t.Error("a mod without an ID cannot be switched off on its own")
	}
	if !p.Entries[1].Enabled("Au.Two") {
		t.Error("a mod absent from Disabled is on")
	}
	if e, m, ok := p.FindMod("", "au.two"); !ok || e.Key != "b" || m.UniqueID != "Au.Two" {
		t.Errorf("FindMod any entry = %v %v %v", e.Key, m.UniqueID, ok)
	}
	if _, _, ok := p.FindMod("a", "Au.Two"); ok {
		t.Error("FindMod with a key must stay in that entry")
	}
}
