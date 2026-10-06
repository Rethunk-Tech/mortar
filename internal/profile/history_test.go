package profile

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func addFarmMod(t *testing.T, e env) Profile {
	t.Helper()
	p := mustCreate(t, e, "Farm")
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestHistoryRecordsEachOperation(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	e.item(t, "local-a2", map[string]string{"manifest.json": `{"Name":"Me.A","Author":"me","Version":"2.0.0","UniqueID":"Me.A"}`})
	p := addFarmMod(t, e)
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "smapi:Me.A", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "smapi:Me.A", true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetPinned("stardew", p.ID, "local-a", true, ""); err != nil {
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
		{Key: "local-a2", ID: "smapi:Me.A"},
		{Key: "local-b", ID: "smapi:Me.B"},
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
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	p := addFarmMod(t, e)
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
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{
		"manifest.json": manifestJSON("Me.A"),
		"config.json":   "shipped",
	})
	p := addFarmMod(t, e)
	afterAdd, err := e.History("stardew", p.ID)
	if err != nil || len(afterAdd) == 0 {
		t.Fatalf("history after add: %v %v", afterAdd, err)
	}
	writeFile(t, e.mods(p.ID), "local-a/config.json", "mine")
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "smapi:Me.A", false); err != nil {
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
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
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
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	// Fill the log to its cap directly: reaching it one profile write at a time costs a full save per event.
	data, err := s.loadHistory("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	seed := HistoryEvent{ID: "seed", SnapshotID: "seed"}
	data.Events = slices.Repeat([]HistoryEvent{seed}, maxHistory)
	if err := writeHistory(data.dir, data, 0); err != nil {
		t.Fatal(err)
	}
	for i := range 20 {
		if _, err := s.update("stardew", p.ID, func(p *Profile, _ string) error {
			p.Entries = []Entry{{Key: "k", Mods: []Component{{ID: "smapi:x", Name: "X", Version: "1", Folder: "."}}}}
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
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	entries := []Entry{{Key: "k", Mods: []Component{{ID: "smapi:x", Name: "X", Version: "1", Folder: "."}}}}
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
	sidecars, err := os.ReadDir(filepath.Join(dir, snapshotsDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Events) != 3 || len(sidecars) != 2 {
		t.Fatalf("history events=%d snapshots=%d, want 3 and 2", len(data.Events), len(sidecars))
	}
	raw, err := fsx.ReadFile(filepath.Join(dir, historyFile))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(`"snapshots"`)) {
		t.Fatal("history.json embeds snapshot bodies")
	}
}

func TestCorruptHistoryIsQuarantined(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
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
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := mustCreate(t, e, "Farm")
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
	t.Parallel()
	e := newEnv(t)
	for _, key := range []string{"a", "b", "c"} {
		e.item(t, key, map[string]string{"manifest.json": manifestJSON(key)})
	}
	p := mustCreate(t, e, "Farm")
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
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
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
	t.Parallel()
	a1 := Entry{Key: "a", Mods: []Component{{ID: "smapi:A.Mod", Name: "Alpha", Version: "1.0"}}}
	a2 := Entry{Key: "a2", Mods: []Component{{ID: "smapi:A.Mod", Name: "Alpha", Version: "2.0"}}}
	b := Entry{Key: "b", Mods: []Component{{ID: "smapi:B.Mod", Name: "Beta", Version: "1.0"}}}
	c := Entry{Key: "c", Mods: []Component{{ID: "smapi:C.Mod", Name: "Gamma", Version: "1.0"}}}
	before := []Entry{a1, b}
	after := []Entry{a2, c}
	added, removed, updated := ModDiffCounts(before, after)
	if added != 1 || removed != 1 || updated != 1 {
		t.Fatalf("got +%d −%d ~%d, want +1 −1 ~1", added, removed, updated)
	}
}

func TestHistoryIncludesModDiffCounts(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	p := addFarmMod(t, e)
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
	t.Parallel()
	a := Entry{Key: "gmcm", Mods: []Component{{ID: "smapi:spacechase0.GenericModConfigMenu", Name: "Generic Mod Config Menu"}}}
	b := Entry{Key: "npc", Mods: []Component{{ID: "smapi:Bouhm.NPCMapLocations", Name: "NPC Map Locations"}}}
	before := []Entry{a, b}
	after := []Entry{a, b}
	after[0].Disabled = []mod.ID{"smapi:spacechase0.GenericModConfigMenu"}
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
	t.Parallel()
	e := newEnv(t)
	e.item(t, "smapi-1.0.0", bundle())
	p := mustCreate(t, e, "Farm")
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
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	a := mustCreate(t, e, "A")
	b := mustCreate(t, e, "B")
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
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
	if _, err := e.RemoveEntry("stardew", p.ID, "local-a"); err != nil {
		t.Fatal(err)
	}
	keys, err := e.StoreKeys(true)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(keys["stardew"], "local-a") {
		t.Fatalf("history key missing from StoreKeys: %v", keys["stardew"])
	}
	dir, err := e.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, snapshotsDir)); err != nil {
		t.Fatal(err)
	}
	cached, err := e.StoreKeys(true)
	if err != nil || !slices.Contains(cached["stardew"], "local-a") {
		t.Fatalf("cached history keys = %v, %v", cached["stardew"], err)
	}
	live, err := e.StoreKeys(false)
	if err != nil || slices.Contains(live["stardew"], "local-a") {
		t.Fatalf("live keys = %v, %v", live["stardew"], err)
	}
}

func TestApplyEntrySnapshotKeepsModsWhenPlaceFails(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
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
	if err := e.applyEntrySnapshot("stardew", &cur, dir, []Entry{{Key: "missing-key", Mods: []Component{{ID: "smapi:x"}}}}); err == nil {
		t.Fatal("expected place to fail")
	}
	if _, err := os.Stat(mods); err != nil {
		t.Fatal("mods/ was wiped after a failed snapshot apply")
	}
}

func TestRevertRestoresLiveModsWhenProfileJSONFails(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
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
	t.Parallel()
	e := newEnv(t)
	alpha := mustCreate(t, e, "Alpha")
	beta := mustCreate(t, e, "Beta")
	hidden := mustCreate(t, e, "Hidden")
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

func TestChangesSinceCachedUntilProfileUpdated(t *testing.T) {
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	entry := Entry{Key: "k", Mods: []Component{{ID: "smapi:x", Name: "X", Version: "1", Folder: "."}}}
	if _, err := s.update("stardew", p.ID, func(p *Profile, _ string) error {
		p.Entries = []Entry{entry}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	since := time.Now().UTC().Add(-time.Hour)
	decodes := 0
	onHistoryDecode = func() { decodes++ }
	t.Cleanup(func() { onHistoryDecode = nil })
	if _, err := s.ChangesSince("stardew", p.ID, since); err != nil {
		t.Fatal(err)
	}
	if decodes == 0 {
		t.Fatal("first ChangesSince did not decode history")
	}
	decodes = 0
	if _, err := s.ChangesSince("stardew", p.ID, since); err != nil {
		t.Fatal(err)
	}
	if decodes != 0 {
		t.Fatalf("cached ChangesSince decoded %d times", decodes)
	}
	if _, err := s.update("stardew", p.ID, func(p *Profile, _ string) error {
		p.Entries[0].Mods[0].Version = "2"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	decodes = 0
	if _, err := s.ChangesSince("stardew", p.ID, since); err != nil {
		t.Fatal(err)
	}
	if decodes == 0 {
		t.Fatal("ChangesSince after update used stale cache")
	}
}

func TestAppendedEventsCarryTheirCounts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	one := []Entry{{Key: "a", Mods: []Component{{ID: "smapi:A.Mod", Version: "1.0"}}}}
	two := append(cloneEntries(one), Entry{Key: "b", Mods: []Component{{ID: "smapi:B.Mod", Version: "1.0"}}})
	for _, after := range [][]Entry{one, two} {
		if _, err := appendHistory(dir, HistoryEvent{Kind: historyPinned, Label: "change"}, after, 0); err != nil {
			t.Fatal(err)
		}
	}
	data, err := readHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !data.Counted || len(data.Events) != 2 {
		t.Fatalf("counted=%v events=%d", data.Counted, len(data.Events))
	}
	if ev := data.Events[1]; ev.Added != 1 || ev.Removed != 0 || ev.Updated != 0 {
		t.Fatalf("second event counts = %+v", ev)
	}
}

func TestPlainSnapshotIsReadAndGzipped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, snapshotsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(dir, snapshotsDir, "abcd.json")
	if err := os.WriteFile(plain, []byte(`[{"key":"a"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, ok := readSnapshotFile(dir, "abcd")
	if !ok || len(entries) != 1 || entries[0].Key != "a" {
		t.Fatalf("entries = %v, %v", entries, ok)
	}
	if _, err := os.Stat(plain); !os.IsNotExist(err) {
		t.Fatalf("plain snapshot still there: %v", err)
	}
	again, ok := readSnapshotFile(dir, "abcd")
	if !ok || len(again) != 1 || again[0].Key != "a" {
		t.Fatalf("gzipped read = %v, %v", again, ok)
	}
}

func TestBeginUpdateBatchIsTheBaselineWithAFreshBatch(t *testing.T) {
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	svc := &Service{store: s}
	a, err := svc.BeginUpdateBatch("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.BeginUpdateBatch("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.Baseline("stardew", p.ID)
	if err != nil || a.Before != before || b.Before != before || a.Batch == "" || a.Batch == b.Batch {
		t.Fatalf("batches %+v %+v, baseline %q: %v", a, b, before, err)
	}
}

func TestAChangeReturnsTheIDOfItsHistoryEventWithoutWritingIt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
	changed, err := e.SetModEnabled("stardew", p.ID, "local-a", "smapi:Me.A", false)
	if err != nil {
		t.Fatal(err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil || len(events) == 0 {
		t.Fatalf("history: %v %v", events, err)
	}
	if changed.LastChange == "" || changed.LastChange != events[0].ID {
		t.Fatalf("LastChange %q, newest event %q", changed.LastChange, events[0].ID)
	}
	_, dir, err := e.readDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := readAt(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if onDisk.LastChange != "" {
		t.Fatal("profile.json holds lastChange")
	}
}
