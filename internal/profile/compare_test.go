package profile

import (
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestDiffProfilesSplitsByUniqueID(t *testing.T) {
	a := Profile{Entries: []Entry{
		{Key: "smapi-1", Source: Source{Kind: SourceSMAPI}, Mods: []Component{{ID: "smapi:SMAPI.ConsoleCommands", Name: "Console", Version: "1"}}},
		{Key: "only-a", Source: Source{Kind: KindLocal, Name: "a.zip"}, Mods: []Component{{ID: "smapi:Me.A", Name: "Alpha", Version: "1.0.0"}}},
		{Key: "shared-old", Source: Source{Kind: KindNexus, ModID: 1, FileID: 1}, Mods: []Component{{ID: "smapi:Me.Shared", Name: "Shared", Version: "1.0.0"}}},
		{Key: "off-a", Source: Source{Kind: KindLocal, Name: "off.zip"}, Mods: []Component{{ID: "smapi:Me.Off", Name: "Off", Version: "1.0.0"}}, Disabled: []mod.ID{"smapi:Me.Off"}},
	}}
	b := Profile{Entries: []Entry{
		{Key: "smapi-1", Source: Source{Kind: SourceSMAPI}, Mods: []Component{{ID: "smapi:SMAPI.ConsoleCommands", Name: "Console", Version: "1"}}},
		{Key: "only-b", Source: Source{Kind: KindLocal, Name: "b.zip"}, Mods: []Component{{ID: "smapi:Me.B", Name: "Beta", Version: "2.0.0"}}},
		{Key: "shared-new", Source: Source{Kind: KindNexus, ModID: 1, FileID: 2}, Mods: []Component{{ID: "smapi:Me.Shared", Name: "Shared", Version: "2.0.0"}}},
		{Key: "off-a", Source: Source{Kind: KindLocal, Name: "off.zip"}, Mods: []Component{{ID: "smapi:Me.Off", Name: "Off", Version: "1.0.0"}}},
	}}
	d := DiffProfiles(a, b)
	if len(d.OnlyA) != 1 || d.OnlyA[0].ID != "smapi:Me.A" || len(d.OnlyB) != 1 || d.OnlyB[0].ID != "smapi:Me.B" {
		t.Fatalf("only = %+v / %+v", d.OnlyA, d.OnlyB)
	}
	if len(d.Changed) != 2 {
		t.Fatalf("changed = %+v", d.Changed)
	}
	byID := map[string]DiffPair{}
	for _, p := range d.Changed {
		byID[p.ID.Local()] = p
	}
	if byID["Me.Shared"].A.Version != "1.0.0" || byID["Me.Shared"].B.Version != "2.0.0" {
		t.Fatalf("version pair = %+v", byID["Me.Shared"])
	}
	if byID["Me.Off"].A.Enabled || !byID["Me.Off"].B.Enabled {
		t.Fatalf("enabled pair = %+v", byID["Me.Off"])
	}
}

func TestCopyModsAddsFromStoreAndRespectsTheRunningLock(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": `{"Name":"B","Author":"me","Version":"2.0.0","UniqueID":"Me.B"}`})
	from := mustCreate(t, e, "From")
	to := mustCreate(t, e, "To")
	if _, err := e.AddEntry("stardew", from.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", from.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", from.ID, "local-b", "smapi:Me.B", false); err != nil {
		t.Fatal(err)
	}

	got, err := e.CopyMods("stardew", from.ID, to.ID, []mod.ID{"smapi:Me.A", "smapi:me.b"})
	if err != nil {
		t.Fatal(err)
	}
	src, err := e.load("stardew", from.ID)
	if err != nil {
		t.Fatal(err)
	}
	d := DiffProfiles(src, got)
	if len(d.OnlyA) != 0 || len(d.OnlyB) != 0 || len(d.Changed) != 0 {
		t.Fatalf("after copy diff = %+v", d)
	}
	off := false
	for _, entry := range got.Entries {
		if entry.Key == "local-b" {
			off = !entry.Enabled("smapi:Me.B")
		}
	}
	if !off {
		t.Fatalf("copied disabled mod as enabled: %+v", got.Entries)
	}

	e.item(t, "local-c", map[string]string{"manifest.json": `{"Name":"C","Author":"me","Version":"1.0.0","UniqueID":"Me.C"}`})
	if _, err := e.AddEntry("stardew", from.ID, "local-c", Source{Kind: KindLocal, Name: "c.zip"}); err != nil {
		t.Fatal(err)
	}
	e.Running = func(_, id string) bool { return id == to.ID }
	var re *RunningError
	if _, err := e.CopyMods("stardew", from.ID, to.ID, []mod.ID{"smapi:Me.C"}); !errors.As(err, &re) {
		t.Fatalf("running dest = %v", err)
	}
}
