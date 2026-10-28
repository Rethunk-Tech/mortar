package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// Service.Mods leaves a deleted entry folder missing (so drift offers Restore) and does not count a switched-off
// ".<key>" copy as missing.
func TestServiceModsKeepsMissingFoldersMissing(t *testing.T) {
	t.Parallel()
	m1, m2 := manifestJSON("me.a"), manifestJSON("me.b")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m1}, map[string]string{"B/manifest.json": m2})
	if _, err := e.AddEntry("stardew", p.ID, "a-2", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(e.Store, t.TempDir(), nil)
	mods := e.mods(p.ID)
	if err := os.RemoveAll(filepath.Join(mods, "a-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Mods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(mods, "a-1")) || !exists(filepath.Join(mods, "a-2")) {
		t.Fatalf("mods/ = %v", names(t, mods))
	}
	drift, err := svc.ScanModsDrift("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 1 || drift[0].Kind != DriftDeleted || drift[0].Key != "a-1" {
		t.Fatalf("drift = %+v", drift)
	}

	if err := os.Rename(filepath.Join(mods, "a-2"), filepath.Join(mods, ".a-2")); err != nil {
		t.Fatal(err)
	}
	missing, err := e.missingEntryFolders("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0] != "a-1" {
		t.Fatalf("missing = %v, want only a-1", missing)
	}
	if _, err := svc.Mods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(mods, ".a-2")) {
		t.Fatalf("hidden copy removed: %v", names(t, mods))
	}
}
