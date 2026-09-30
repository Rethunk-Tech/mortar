package profile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

func TestClassifyHistoryNamesTheEntryThatChanged(t *testing.T) {
	a := Entry{Key: "gmcm", Mods: []EntryMod{{UniqueID: "spacechase0.GenericModConfigMenu", Name: "Generic Mod Config Menu"}}}
	b := Entry{Key: "npc", Mods: []EntryMod{{UniqueID: "Bouhm.NPCMapLocations", Name: "NPC Map Locations"}}}
	before := []Entry{a, b}
	after := []Entry{a, b}
	after[0].Disabled = []string{"spacechase0.GenericModConfigMenu"}
	got := classifyHistory(before, after)
	if got.Kind != historyDisabled || got.Label != "Disabled Generic Mod Config Menu" {
		t.Fatalf("got %+v", got)
	}
	got = classifyHistory(after, before)
	if got.Kind != historyEnabled || got.Label != "Enabled Generic Mod Config Menu" {
		t.Fatalf("re-enable got %+v", got)
	}
}

func TestApplyBundledDoesNotRecordHistory(t *testing.T) {
	e := newEnv(t)
	e.item(t, "smapi-1.0.0", bundle())
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ApplyBundled("stardew", smapiBundle("smapi-1.0.0")); err != nil {
		t.Fatal(err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("bundled bookkeeping recorded history: %+v", events)
	}
}

func TestHistoryQuietIsPerProfile(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	a, err := e.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Create("stardew", "B")
	if err != nil {
		t.Fatal(err)
	}
	e.setHistoryQuiet(a.ID, true)
	if _, err := e.AddEntry("stardew", a.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", b.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	ha, err := e.History("stardew", a.ID)
	if err != nil || len(ha) != 0 {
		t.Fatalf("quiet profile history = %+v, %v", ha, err)
	}
	hb, err := e.History("stardew", b.ID)
	if err != nil || len(hb) == 0 {
		t.Fatalf("other profile history = %+v, %v", hb, err)
	}
}

func TestStoreKeysIncludesHistorySnapshots(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RemoveEntry("stardew", p.ID, "local-a"); err != nil {
		t.Fatal(err)
	}
	keys, err := e.StoreKeys()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(keys["stardew"], "local-a") {
		t.Fatalf("history key missing from StoreKeys: %v", keys["stardew"])
	}
}

func TestApplyEntrySnapshotKeepsModsWhenPlaceFails(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	dir, err := e.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	mods := filepath.Join(dir, "mods", "local-a")
	if _, err := os.Stat(mods); err != nil {
		t.Fatal(err)
	}
	cur, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.applyEntrySnapshot("stardew", &cur, dir, []Entry{{Key: "missing-key", Mods: []EntryMod{{UniqueID: "x"}}}}); err == nil {
		t.Fatal("expected place to fail")
	}
	if _, err := os.Stat(mods); err != nil {
		t.Fatal("mods/ was wiped after a failed snapshot apply")
	}
}

func TestRevertRestoresLiveModsWhenProfileJSONFails(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	mods, err := e.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var cfg string
	err = filepath.WalkDir(mods, func(fp string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		if d.Name() == "manifest.json" {
			cfg = filepath.Join(filepath.Dir(fp), "config.json")
			return os.WriteFile(cfg, []byte(`{"keep":true}`), 0o600)
		}
		return nil
	})
	if err != nil || cfg == "" {
		t.Fatalf("seed config.json: %v", err)
	}
	_, err = e.updateLocked("stardew", p.ID, func(p *Profile, dir string) error {
		if err := e.applyEntrySnapshot("stardew", p, dir, nil); err != nil {
			return err
		}
		path := filepath.Join(dir, fileName)
		if err := os.Remove(path); err != nil {
			return err
		}
		return os.Mkdir(path, 0o700)
	})
	if err == nil {
		t.Fatal("expected profile.json write to fail")
	}
	if b, readErr := os.ReadFile(filepath.Clean(cfg)); readErr != nil || string(b) != `{"keep":true}` {
		t.Fatalf("live config lost: %q, %v", b, readErr)
	}
}
