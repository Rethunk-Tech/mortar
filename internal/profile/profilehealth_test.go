package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// healthEnv is a profile with one installed mod, its drift baseline recorded.
func healthEnv(t *testing.T) (env, *Service, Profile) {
	t.Helper()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	zip := buildZip(t, "mod.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	if _, err := e.InstallArchive("stardew", p.ID, zip); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ScanModsDrift("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	p, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return e, &Service{store: e.Store}, p
}

func findingsOf(t *testing.T, svc *Service, id, kind string) []HealthFinding {
	t.Helper()
	all, err := svc.ProfileHealth("stardew", id)
	if err != nil {
		t.Fatal(err)
	}
	var out []HealthFinding
	for _, f := range all {
		if f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

func TestHealthMissingStoreItem(t *testing.T) {
	t.Parallel()
	e, svc, p := healthEnv(t)
	if got := findingsOf(t, svc, p.ID, HealthMissing); len(got) != 0 {
		t.Fatalf("healthy profile: %#v", got)
	}
	path, err := e.items.Path("stardew", p.Entries[0].Key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	got := findingsOf(t, svc, p.ID, HealthMissing)
	// A local archive has no source to download from, so there is nothing to repair with.
	if len(got) != 1 || got[0].Repair != "" || len(got[0].Entries) != 1 || got[0].Entries[0].Key != p.Entries[0].Key {
		t.Fatalf("missing = %#v", got)
	}
}

func TestHealthDriftRestoresFromStore(t *testing.T) {
	t.Parallel()
	e, svc, p := healthEnv(t)
	if err := os.RemoveAll(filepath.Join(e.mods(p.ID), p.Entries[0].Key)); err != nil {
		t.Fatal(err)
	}
	got := findingsOf(t, svc, p.ID, HealthDrift)
	if len(got) != 1 || got[0].Repair != RepairRestore || got[0].Cause != string(DriftDeleted) {
		t.Fatalf("drift = %#v", got)
	}
	before, _ := e.History("stardew", p.ID)
	if _, err := svc.RepairProfile("stardew", p.ID, []string{got[0].ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), p.Entries[0].Key, "A", "manifest.json")); err != nil {
		t.Fatalf("not restored: %v", err)
	}
	if left := findingsOf(t, svc, p.ID, HealthDrift); len(left) != 0 {
		t.Fatalf("after repair: %#v", left)
	}
	after, _ := e.History("stardew", p.ID)
	if len(after) != len(before)+1 || after[0].Kind != historyRestored {
		t.Fatalf("history = %#v", after)
	}
}

func TestHealthUnusedStoreItems(t *testing.T) {
	t.Parallel()
	e, svc, p := healthEnv(t)
	e.item(t, "orphan", map[string]string{"O/manifest.json": manifestJSON("X.O")})
	svc.HealthKeep = func() (map[string][]string, error) { return e.StoreKeys(true) }
	got := findingsOf(t, svc, p.ID, HealthUnused)
	if len(got) != 1 || len(got[0].Items) != 1 || got[0].Repair != RepairCleanup {
		t.Fatalf("unused = %#v", got)
	}
	// Cleanup is the app's storage dialog; a repair here must not delete anything.
	if _, err := svc.RepairProfile("stardew", p.ID, []string{got[0].ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.items.Path("stardew", "orphan"); err != nil {
		t.Fatalf("orphan deleted: %v", err)
	}
}

func TestHealthUnreadableSnapshotIsDropped(t *testing.T) {
	t.Parallel()
	e, svc, p := healthEnv(t)
	events := e.mustHistory(t, p.ID)
	broken := events[0]
	path, _ := snapshotFilePath(filepath.Dir(e.mods(p.ID)), broken.SnapshotID)
	if err := os.WriteFile(path, []byte("not gzip"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := findingsOf(t, svc, p.ID, HealthSnapshot)
	if len(got) != 1 || got[0].ID != HealthSnapshot+":"+broken.ID || got[0].Cause != "unreadable" || !got[0].At.Equal(broken.At) {
		t.Fatalf("snapshot = %#v", got)
	}
	repaired, err := svc.RepairProfile("stardew", p.ID, []string{got[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if !entriesEqual(repaired.Entries, p.Entries) {
		t.Fatal("dropping a snapshot changed the profile")
	}
	after, _ := e.History("stardew", p.ID)
	if len(after) != len(events)-1 {
		t.Fatalf("history = %#v", after)
	}
}

func TestHealthLeftoverJournalRecovers(t *testing.T) {
	t.Parallel()
	_, svc, p := healthEnv(t)
	journals := []string{"/journal/install-1"}
	svc.HealthJournals = func(string) []string { return journals }
	svc.HealthRecover = func(string) error { journals = nil; return nil }
	got := findingsOf(t, svc, p.ID, HealthJournal)
	if len(got) != 1 || got[0].Repair != RepairRecover {
		t.Fatalf("journal = %#v", got)
	}
	if _, err := svc.RepairProfile("stardew", p.ID, []string{got[0].ID}); err != nil {
		t.Fatal(err)
	}
	if left := findingsOf(t, svc, p.ID, HealthJournal); len(left) != 0 {
		t.Fatalf("after recover: %#v", left)
	}
}

func TestHealthRevertKeepsConfigAndData(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	zip := buildZip(t, "mod.zip", map[string]string{
		"A/manifest.json": manifestJSON("X.A"),
		"A/assets/x.json": "{}",
	})
	if _, err := e.InstallArchive("stardew", p.ID, zip); err != nil {
		t.Fatal(err)
	}
	p, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(e.mods(p.ID), p.Entries[0].Key, "A")
	writeTimed(t, filepath.Join(dir, "config.json"), `{"mine":true}`, time.Time{})
	writeTimed(t, filepath.Join(dir, "data", "farm-1.json"), `{"gold":1}`, time.Time{})
	if _, err := e.ScanModsDrift("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	writeTimed(t, filepath.Join(dir, "assets", "x.json"), `{"edited":true}`, time.Now().Add(time.Hour))
	svc := &Service{store: e.Store}
	got := findingsOf(t, svc, p.ID, HealthDrift)
	if len(got) != 1 || got[0].Cause != string(DriftChanged) || got[0].Repair != RepairRevert {
		t.Fatalf("drift = %#v", got)
	}
	if _, err := svc.RepairProfile("stardew", p.ID, []string{got[0].ID}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"config.json", "data/farm-1.json"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s lost: %v", rel, err)
		}
	}
	b, err := fsx.ReadFile(filepath.Join(dir, "assets", "x.json"))
	if err != nil || string(b) != "{}" {
		t.Fatalf("not reverted: %s %v", b, err)
	}
}

func TestRestoreLabelNamesWhatWasRestored(t *testing.T) {
	t.Parallel()
	e, svc, p := healthEnv(t)
	zip := buildZip(t, "b.zip", map[string]string{"B/manifest.json": manifestJSON("X.B")})
	if _, err := e.InstallArchive("stardew", p.ID, zip); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ScanModsDrift("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	p, err := e.read("stardew", p.ID)
	if err != nil || len(p.Entries) != 2 {
		t.Fatalf("entries = %v, %v", p.Entries, err)
	}
	for _, en := range p.Entries {
		if err := os.RemoveAll(filepath.Join(e.mods(p.ID), en.Key)); err != nil {
			t.Fatal(err)
		}
	}
	// The first mod's store copy is gone too, so its restore fails and only the second is restored.
	path, err := e.items.Path("stardew", p.Entries[0].Key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, f := range findingsOf(t, svc, p.ID, HealthDrift) {
		ids = append(ids, f.ID)
	}
	_, _ = svc.RepairProfile("stardew", p.ID, ids)
	hist, _ := e.History("stardew", p.ID)
	if len(hist) == 0 || hist[0].Kind != historyRestored || hist[0].Change != ChangeRestoredFromStore ||
		hist[0].Name != entryLabel(p.Entries[1]) {
		t.Fatalf("history = %#v", hist)
	}
}
