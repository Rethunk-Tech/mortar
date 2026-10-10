package profile

import (
	"path/filepath"
	"testing"
	"time"
)

func TestHeldCopiesExpireWithTheTrashRetention(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "Mods/mc/mc_settings.cfg", "player=3")
	writeFile(t, dir, "Mods/mc/notes.cfg", "mine")
	held := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, rel := range []string{"Mods/mc/mc_settings.cfg", "Mods/mc/notes.cfg"} {
		if err := holdChanged(dir, rel); err != nil {
			t.Fatal(err)
		}
	}
	if err := markHeld(dir, "Mods/mc/mc_settings.cfg", held, true); err != nil {
		t.Fatal(err)
	}
	if err := markHeld(dir, "Mods/mc/notes.cfg", held.Add(20*24*time.Hour), true); err != nil {
		t.Fatal(err)
	}
	// A copy with no record of when it was held, as a restored backup brings.
	writeFile(t, dir, "changed/Mods/other/restored.cfg", "restored")
	keep := 30 * 24 * time.Hour

	if n, err := expireHeld(dir, keep, held.Add(29*24*time.Hour)); err != nil || n != 0 {
		t.Fatalf("inside the retention: expired %d, %v", n, err)
	}
	n, err := expireHeld(dir, keep, held.Add(31*24*time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("past the retention of the oldest: expired %d, %v; want 1", n, err)
	}
	if exists(filepath.Join(dir, "changed", "Mods", "mc", "mc_settings.cfg")) {
		t.Fatal("the copy held past the retention is still there")
	}
	for _, rel := range []string{"changed/Mods/mc/notes.cfg", "changed/Mods/other/restored.cfg"} {
		if !exists(filepath.Join(dir, filepath.FromSlash(rel))) {
			t.Fatalf("%s was deleted before its own retention ran out", rel)
		}
	}
	if ok, err := restoreChanged(dir, "Mods/mc/notes.cfg"); err != nil || !ok {
		t.Fatalf("restore = %v, %v", ok, err)
	}
	if _, known := readHeld(dir)["Mods/mc/notes.cfg"]; known {
		t.Fatal("a copy that came back is still counted as held")
	}
}
