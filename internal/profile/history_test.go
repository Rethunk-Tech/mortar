package profile

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestHistoryRecordsEachOperation(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	e.item(t, "local-a2", map[string]string{"manifest.json": `{"Name":"Me.A","Author":"me","Version":"2.0.0","UniqueID":"Me.A"}`})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "Me.A", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "Me.A", true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetPinned("stardew", p.ID, "local-a", true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.UpdateEntry("stardew", p.ID, "local-a", "local-a2"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RemoveEntry("stardew", p.ID, "local-b"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModsEnabled("stardew", p.ID, []EnableRef{
		{Key: "local-a2", UniqueID: "Me.A"},
		{Key: "local-b", UniqueID: "Me.B"},
	}, false); err != nil {
		t.Fatal(err)
	}
	if err := e.recordSnapshot("stardew", p.ID, historyImported, "Imported 2 mods", 2); err != nil {
		t.Fatal(err)
	}
	if err := e.recordSnapshot("stardew", p.ID, historyRestored, "Restored 2 mods", 2); err != nil {
		t.Fatal(err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		historyRestored, historyImported, historyBulk, historyAdded, historyRemoved,
		historyUpdated, historyPinned, historyEnabled, historyDisabled, historyAdded, historyAdded,
	}
	if len(events) < len(want) {
		t.Fatalf("got %d events, want at least %d: %+v", len(events), len(want), kinds(events))
	}
	got := kinds(events)
	for i, k := range want {
		if got[i] != k {
			t.Fatalf("event %d kind = %q, want %q (all %v)", i, got[i], k, got)
		}
	}
}

func kinds(events []HistoryEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.Kind
	}
	return out
}

func TestHistoryRevertRestoresEntries(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	afterAdd, err := e.History("stardew", p.ID)
	if err != nil || len(afterAdd) == 0 {
		t.Fatalf("history after add: %v %v", afterAdd, err)
	}
	snap := cloneEntries(afterAdd[0].Entries)
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	reverted, err := e.Revert("stardew", p.ID, afterAdd[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reverted.Entries, snap) {
		t.Fatalf("entries after revert = %+v, want %+v", reverted.Entries, snap)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil || len(events) == 0 || events[0].Kind != historyReverted {
		t.Fatalf("revert event = %+v, %v", events, err)
	}
}

func TestHistoryRevertMissingStoreKeys(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil || len(events) == 0 {
		t.Fatal(err)
	}
	if err := e.items.Remove([]store.Ref{{Game: "stardew", Key: "local-a"}}); err != nil {
		t.Fatal(err)
	}
	_, err = e.Revert("stardew", p.ID, events[0].ID)
	if missing, ok := errors.AsType[*MissingKeys](err); !ok || len(missing.Keys) == 0 {
		t.Fatalf("Revert = %v, want MissingKeys", err)
	}
}

func TestHistoryBounded(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	for i := range maxHistory + 20 {
		if _, err := s.update("stardew", p.ID, func(p *Profile, _ string) error {
			p.Entries = []Entry{{Key: "k", Mods: []EntryMod{{UniqueID: "x", Name: "X", Version: "1", Folder: "."}}}}
			p.Entries[0].Mods[0].Version = "1"
			if i%2 == 0 {
				p.Entries[0].Pinned = true
			} else {
				p.Entries[0].Pinned = false
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := s.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != maxHistory {
		t.Fatalf("len = %d, want %d", len(events), maxHistory)
	}
}

func TestHistoryRevertRefusedWhileRunning(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil || len(events) == 0 {
		t.Fatal(err)
	}
	e.Running = func(game, id string) bool { return game == "stardew" && id == p.ID }
	_, err = e.Revert("stardew", p.ID, events[0].ID)
	if _, ok := errors.AsType[*RunningError](err); !ok {
		t.Fatalf("Revert = %v, want RunningError", err)
	}
}
