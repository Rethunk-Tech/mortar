package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// Keeping a drift scan, forgetting an entry and removing then restoring an untracked folder all act on the profile
// and its recorded snapshot, so the next scan agrees with what the player chose.
func TestDriftKeepForgetRemoveRestore(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	svc, _ := e.withSettings(t)
	if got := e.DriftBaseline("stardew", "no-such-profile"); got != "" {
		t.Fatalf("baseline of a missing profile = %q", got)
	}
	p := mustCreate(t, e, "Farm")
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if got := e.mustScanDrift(t, p.ID); len(got) != 0 {
		t.Fatalf("after Mortar's own install: %#v", got)
	}
	baseline := e.DriftBaseline("stardew", p.ID)
	if baseline == "" {
		t.Fatal("a recorded snapshot has a baseline")
	}
	key := "local-a"

	writeFile(t, e.mods(p.ID), filepath.Join(key, "manifest.json"), "edited outside Mortar")
	got := e.mustScanDrift(t, p.ID)
	if len(got) != 1 || got[0].Kind != DriftChanged || got[0].Key != key {
		t.Fatalf("after an outside edit: %#v", got)
	}
	if err := svc.KeepDriftChanges("stardew", p.ID, key); err != nil {
		t.Fatal(err)
	}
	if got := e.mustScanDrift(t, p.ID); len(got) != 0 {
		t.Fatalf("after keeping the change: %#v", got)
	}

	if _, err := svc.ForgetDriftEntry("stardew", p.ID, key); err != nil {
		t.Fatal(err)
	}
	if prof, err := e.read("stardew", p.ID); err != nil || EntryIndex(prof.Entries, key) >= 0 {
		t.Fatalf("forgotten entry still listed: %+v, %v", prof.Entries, err)
	}

	writeFile(t, e.mods(p.ID), "Loose/a.txt", "player folder")
	token, err := svc.RemoveDriftFolder("stardew", p.ID, "Loose")
	if err != nil || token == "" {
		t.Fatalf("remove = %q, %v", token, err)
	}
	loose := filepath.Join(e.mods(p.ID), "Loose")
	if _, err := os.Stat(loose); err == nil {
		t.Fatal("folder still in mods")
	}
	if err := svc.RestoreDriftFolder("stardew", p.ID, token); err != nil {
		t.Fatal(err)
	}
	if got := testfs.Read(t, filepath.Join(loose, "a.txt")); got != "player folder" {
		t.Fatalf("restored file = %q", got)
	}
}
