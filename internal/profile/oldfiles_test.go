package profile

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestUpdateOldFilesFollowTheSetting(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	v1 := map[string]string{"A/manifest.json": m, "A/gone.json": "g", "A/sub/old.dll": "d"}
	v2 := map[string]string{"A/manifest.json": m, "A/new.json": "n"}
	for _, tc := range []struct {
		mode      string
		answer    *bool
		wantFiles bool
	}{
		{mode: settings.OldFilesDelete},
		{mode: settings.OldFilesKeep, wantFiles: true},
		{mode: settings.OldFilesAsk, answer: new(true), wantFiles: true},
		{mode: settings.OldFilesAsk, answer: new(false)},
	} {
		e, p := updEnv(t, v1, v2)
		e.OldFilesMode = func(string) string { return tc.mode }
		if _, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2"); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(e.mods(p.ID), "a-2", "A")
		pending, err := e.PendingOldFiles("stardew", p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if tc.answer != nil {
			if len(pending) != 1 || pending[0].Key != "a-2" || len(pending[0].Files) != 2 ||
				pending[0].Files[0] != (OldFile{ID: "smapi:me.a", Path: "gone.json"}) {
				t.Fatalf("%s: pending = %+v", tc.mode, pending)
			}
			if exists(filepath.Join(dir, "gone.json")) {
				t.Fatalf("%s: old file left in the mod before the answer", tc.mode)
			}
			if err := e.ResolveOldFiles("stardew", p.ID, "a-2", *tc.answer); err != nil {
				t.Fatal(err)
			}
			if pending, _ = e.PendingOldFiles("stardew", p.ID); len(pending) != 0 {
				t.Fatalf("%s: still pending %+v", tc.mode, pending)
			}
		} else if len(pending) != 0 {
			t.Fatalf("%s: pending = %+v", tc.mode, pending)
		}
		for _, rel := range []string{"gone.json", "sub/old.dll"} {
			if got := exists(filepath.Join(dir, filepath.FromSlash(rel))); got != tc.wantFiles {
				t.Errorf("%s answer %v: %s present = %v", tc.mode, tc.answer, rel, got)
			}
		}
		if read(t, filepath.Join(dir, "new.json")) != "n" {
			t.Errorf("%s: new.json missing", tc.mode)
		}
	}
}

func TestPendingOldFilesDropsSetsOfReplacedVersions(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m, "A/gone.json": "g"}, map[string]string{"A/manifest.json": m})
	if _, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RollBack("stardew", p.ID, "a-2"); err != nil {
		t.Fatal(err)
	}
	pending, err := e.PendingOldFiles("stardew", p.ID)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending = %+v, %v", pending, err)
	}
	if exists(filepath.Join(e.root, "stardew", p.ID, oldFilesDir, "a-2")) {
		t.Fatal("stale set kept")
	}
}

func TestTrashedOldFilesComeBack(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m, "A/gone.json": "g"}, map[string]string{"A/manifest.json": m})
	e.OldFilesMode = func(string) string { return settings.OldFilesAsk }
	if _, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2"); err != nil {
		t.Fatal(err)
	}
	token, err := e.TrashOldFiles("stardew", p.ID, "a-2")
	if err != nil || token == "" {
		t.Fatalf("trash = %q, %v", token, err)
	}
	if pending, _ := e.PendingOldFiles("stardew", p.ID); len(pending) != 0 {
		t.Fatalf("still pending %+v", pending)
	}
	if err := e.RestoreOldFiles("stardew", p.ID, token); err != nil {
		t.Fatal(err)
	}
	if pending, _ := e.PendingOldFiles("stardew", p.ID); len(pending) != 1 || len(pending[0].Files) != 1 {
		t.Fatalf("pending = %+v", pending)
	}
}
