package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTrimHistoryDropsOldEventsAndRecordsIt(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
	for range 3 {
		if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "Me.A", false); err != nil {
			t.Fatal(err)
		}
		if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "Me.A", true); err != nil {
			t.Fatal(err)
		}
	}
	before, err := e.History("stardew", p.ID)
	if err != nil || len(before) < 6 {
		t.Fatalf("history = %d events, %v", len(before), err)
	}
	usage, err := e.TrimHistory("stardew", p.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	after, err := e.History("stardew", p.ID)
	if err != nil || len(after) != 3 || usage.Events != 3 {
		t.Fatalf("after trim: %d events, usage %+v, %v", len(after), usage, err)
	}
	if after[0].Kind != historyTrimmed || after[0].Count != len(before)-2 {
		t.Fatalf("newest event = %+v", after[0])
	}
	dir, _ := e.profileDir("stardew", p.ID)
	snaps, _ := os.ReadDir(filepath.Join(dir, snapshotsDir))
	if len(snaps) > 3 {
		t.Fatalf("%d snapshot files left for 3 events", len(snaps))
	}
	if _, err := e.TrimHistory("stardew", p.ID, 0); err == nil {
		t.Fatal("trim to zero accepted")
	}
}

func TestMigrateHistoryGzipsPlainSnapshotsAndFillsCounts(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
	dir, _ := e.profileDir("stardew", p.ID)
	data, err := readHistory(dir)
	if err != nil || len(data.Events) == 0 {
		t.Fatalf("history: %d events, %v", len(data.Events), err)
	}
	snap := data.Events[0].SnapshotID
	gz := filepath.Join(dir, snapshotsDir, snap+snapshotExt)
	entries, ok := readSnapshotFile(dir, snap)
	if !ok {
		t.Fatal("snapshot unreadable")
	}
	if err := os.Remove(gz); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(dir, snapshotsDir, snap+".json")
	if err := writePlainJSON(plain, entries); err != nil {
		t.Fatal(err)
	}
	data.Counted = false
	if err := writeHistory(dir, data, 0); err != nil {
		t.Fatal(err)
	}
	// writeHistory leaves the plain file in place only until a prune sees it, so put it back to be sure it is plain.
	_ = os.Remove(gz)
	if err := writePlainJSON(plain, entries); err != nil {
		t.Fatal(err)
	}
	n, c, err := e.MigrateHistory()
	if err != nil || n != 1 || c != 1 {
		t.Fatalf("MigrateHistory = %d, %d, %v", n, c, err)
	}
	if _, err := os.Stat(gz); err != nil {
		t.Fatalf("gzipped snapshot missing: %v", err)
	}
	if _, err := os.Stat(plain); err == nil {
		t.Fatal("plain snapshot left behind")
	}
	if n, c, err := e.MigrateHistory(); err != nil || n != 0 || c != 0 {
		t.Fatalf("second pass = %d, %d, %v", n, c, err)
	}
}

func writePlainJSON(path string, entries []Entry) error {
	b, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
