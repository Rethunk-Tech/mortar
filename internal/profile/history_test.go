package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
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
	snap, err := e.Snapshot("stardew", p.ID, afterAdd[0].SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
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

func TestHistoryRevertCarriesModifiedConfig(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{
		"manifest.json": manifestJSON("Me.A"),
		"config.json":   "shipped",
	})
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
	writeFile(t, e.mods(p.ID), "local-a/config.json", "mine")
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "Me.A", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Revert("stardew", p.ID, afterAdd[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(e.mods(p.ID), "local-a", "config.json")); got != "mine" {
		t.Fatalf("config after revert = %q, want mine", got)
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
	missing, ok := errors.AsType[*MissingKeys](err)
	if !ok || len(missing.Keys) == 0 {
		t.Fatalf("Revert = %v, want MissingKeys", err)
	}
	if !strings.Contains(err.Error(), "Me.A") {
		t.Fatalf("Revert error %q, want named mod", err)
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

func TestHistoryDeduplicatesSnapshots(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{Key: "k", Mods: []EntryMod{{UniqueID: "x", Name: "X", Version: "1", Folder: "."}}}}
	for _, version := range []string{"1", "2", "1"} {
		next := cloneEntries(entries)
		next[0].Mods[0].Version = version
		if _, err := s.update("stardew", p.ID, func(p *Profile, _ string) error {
			p.Entries = next
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	dir, err := s.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := readHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Events) != 3 || len(data.Snapshots) != 2 {
		t.Fatalf("history events=%d snapshots=%d, want 3 and 2", len(data.Events), len(data.Snapshots))
	}
}

func TestHistoryMigratesLegacyEntriesInPlace(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{Key: "legacy", Mods: []EntryMod{{UniqueID: "legacy.mod", Folder: "."}}}}
	old := legacyHistoryFileData{Events: []legacyHistoryEvent{{
		ID: "legacy-event", At: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Kind: historyAdded, Label: "Added legacy", Count: 1, Entries: entries,
	}}}
	if err := datadir.WriteJSON(filepath.Join(dir, historyFile), old); err != nil {
		t.Fatal(err)
	}
	events, err := s.History("stardew", p.ID)
	if err != nil || len(events) != 1 {
		t.Fatalf("migrated history = %+v, %v", events, err)
	}
	if events[0].SnapshotID == "" {
		t.Fatal("legacy event has no snapshot ID")
	}
	snapshot, err := s.Snapshot("stardew", p.ID, events[0].ID)
	if err != nil || !reflect.DeepEqual(snapshot, entries) {
		t.Fatalf("migrated snapshot = %+v, %v", snapshot, err)
	}
	data, err := readHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Snapshots) != 1 || len(data.Events) != 1 {
		t.Fatalf("migrated data = %+v", data)
	}
	raw, err := fsx.ReadFile(filepath.Join(dir, historyFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"entries"`) {
		t.Fatalf("legacy entries remain in migrated history: %s", raw)
	}
}

func TestCorruptHistoryIsQuarantined(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, historyFile)
	if err := fsx.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	data, err := readHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Events) != 0 || len(data.Snapshots) != 0 {
		t.Fatalf("empty history = %+v", data)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("corrupt history still exists: %v", err)
	}
	matches, err := filepath.Glob(path + ".corrupt-*")
	if err != nil || len(matches) != 1 {
		t.Fatalf("quarantined history = %v, %v", matches, err)
	}
}

func TestProfileUpdateKeepsFilesWhenHistoryFails(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := e.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, historyFile), 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Key != "local-a" {
		t.Fatalf("profile after history failure = %+v", got)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), "local-a")); err != nil {
		t.Fatalf("installed entry was rolled back: %v", err)
	}
}

func TestHistoryBatchRecordsOneUpdatedSnapshot(t *testing.T) {
	e := newEnv(t)
	for _, key := range []string{"a", "b", "c"} {
		e.item(t, key, map[string]string{"manifest.json": manifestJSON(key)})
	}
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.OpenHistoryBatch("stardew", p.ID, "batch-1"); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"a", "b", "c"} {
		if _, err := e.AddEntry("stardew", p.ID, key, Source{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.CloseHistoryBatch("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Kind != historyBulk || events[0].Count != 3 {
		t.Fatalf("batch history = %+v", events)
	}
	snapshot, err := e.Snapshot("stardew", p.ID, events[0].SnapshotID)
	if err != nil || len(snapshot) != 3 {
		t.Fatalf("batch snapshot = %+v, %v", snapshot, err)
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

func TestModDiffCounts(t *testing.T) {
	a1 := Entry{Key: "a", Mods: []EntryMod{{UniqueID: "A.Mod", Name: "Alpha", Version: "1.0"}}}
	a2 := Entry{Key: "a2", Mods: []EntryMod{{UniqueID: "A.Mod", Name: "Alpha", Version: "2.0"}}}
	b := Entry{Key: "b", Mods: []EntryMod{{UniqueID: "B.Mod", Name: "Beta", Version: "1.0"}}}
	c := Entry{Key: "c", Mods: []EntryMod{{UniqueID: "C.Mod", Name: "Gamma", Version: "1.0"}}}
	before := []Entry{a1, b}
	after := []Entry{a2, c}
	added, removed, updated := ModDiffCounts(before, after)
	if added != 1 || removed != 1 || updated != 1 {
		t.Fatalf("got +%d −%d ~%d, want +1 −1 ~1", added, removed, updated)
	}
}

func TestHistoryIncludesModDiffCounts(t *testing.T) {
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
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 2 {
		t.Fatalf("events: %+v", events)
	}
	// Newest first: second add should show +1 with no removes or updates.
	if events[0].Added != 1 || events[0].Removed != 0 || events[0].Updated != 0 {
		t.Fatalf("latest event counts = +%d −%d ~%d, want +1 −0 ~0", events[0].Added, events[0].Removed, events[0].Updated)
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

func seedHistory(t *testing.T, e env, id string, events []HistoryEvent) {
	t.Helper()
	dir, err := e.profileDir("stardew", id)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeHistory(dir, historyFileData{Events: events, Snapshots: map[string][]Entry{}}, 0); err != nil {
		t.Fatal(err)
	}
}

func TestRecentHistoryOrdersAndSkipsDamaged(t *testing.T) {
	e := newEnv(t)
	alpha, err := e.Create("stardew", "Alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := e.Create("stardew", "Beta")
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := e.Create("stardew", "Hidden")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetHidden("stardew", hidden.ID, true); err != nil {
		t.Fatal(err)
	}
	badID := "0123456789abcdef"
	badDir := filepath.Join(e.root, "stardew", badID)
	if err := os.MkdirAll(filepath.Join(badDir, "mods"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, fileName), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	t3 := t2.Add(time.Hour)
	seedHistory(t, e, alpha.ID, []HistoryEvent{
		{ID: "a-old", At: t1, Kind: historyAdded, Label: "Added old"},
		{ID: "a-new", At: t3, Kind: historyAdded, Label: "Added new"},
	})
	seedHistory(t, e, beta.ID, []HistoryEvent{
		{ID: "b-mid", At: t2, Kind: historyAdded, Label: "Added mid"},
	})
	seedHistory(t, e, hidden.ID, []HistoryEvent{
		{ID: "h-skip", At: t3.Add(time.Hour), Kind: historyAdded, Label: "Added hidden"},
	})
	if err := writeHistory(badDir, historyFileData{
		Events:    []HistoryEvent{{ID: "d-skip", At: t3.Add(2 * time.Hour), Kind: historyAdded, Label: "Added damaged"}},
		Snapshots: map[string][]Entry{},
	}, 0); err != nil {
		t.Fatal(err)
	}

	got, err := e.RecentHistory("stardew")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a-new", "b-mid", "a-old"}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("event %d = %q (%s), want %q", i, got[i].ID, got[i].ProfileName, id)
		}
	}
	if got[0].ProfileName != "Alpha" || got[1].ProfileName != "Beta" {
		t.Fatalf("profile names = %q %q", got[0].ProfileName, got[1].ProfileName)
	}

	many := make([]HistoryEvent, recentHistoryCap+1)
	for i := range many {
		many[i] = HistoryEvent{
			ID: fmt.Sprintf("c%02d", i), At: t1.Add(time.Duration(i) * time.Minute),
			Kind: historyAdded, Label: "Added",
		}
	}
	seedHistory(t, e, alpha.ID, many)
	seedHistory(t, e, beta.ID, nil)
	capped, err := e.RecentHistory("stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(capped) != recentHistoryCap {
		t.Fatalf("capped = %d, want %d", len(capped), recentHistoryCap)
	}
	if capped[0].ID != fmt.Sprintf("c%02d", recentHistoryCap) {
		t.Fatalf("newest after cap = %q", capped[0].ID)
	}
}
