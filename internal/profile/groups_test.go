package profile

import (
	"slices"
	"testing"
)

func TestSetGroupEnabledOneHistoryEvent(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"B/manifest.json": manifestJSON("Me.B")})
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	if p, err = e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if p, err = e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	ka, kb := p.Entries[0].Key, p.Entries[1].Key
	if p, err = e.CreateGroup("stardew", p.ID, "Core"); err != nil {
		t.Fatal(err)
	}
	if p, err = e.AddToGroup("stardew", p.ID, "Core", ka); err != nil {
		t.Fatal(err)
	}
	if p, err = e.AddToGroup("stardew", p.ID, "Core", kb); err != nil {
		t.Fatal(err)
	}
	before, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.SetGroupEnabled("stardew", p.ID, "Core", false)
	if err != nil {
		t.Fatal(err)
	}
	after, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("history events = %d, want %d", len(after), len(before)+1)
	}
	for _, en := range p.Entries {
		if en.Key == ka || en.Key == kb {
			if len(en.Disabled) == 0 {
				t.Fatalf("entry %s still enabled", en.Key)
			}
		}
	}
}

func TestRemoveEntryDropsGroupKeys(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	if p, err = e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	key := p.Entries[0].Key
	if p, err = e.AddToGroup("stardew", p.ID, "Core", key); err != nil {
		t.Fatal(err)
	}
	p, err = e.RemoveEntry("stardew", p.ID, key)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(p.Groups, func(g Group) bool { return slices.Contains(g.Keys, key) }) {
		t.Fatalf("groups still name %s: %+v", key, p.Groups)
	}
}
