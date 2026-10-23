package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// The Sims 4 names its user folder after the game's language, and switching language starts a new one beside the old.
func TestLocalizedFolderIsTheOneChangedLast(t *testing.T) {
	ea := filepath.Join(t.TempDir(), "Electronic Arts")
	names := map[string][]string{"The Sims 4": {"Die Sims 4", "Les Sims 4"}}
	p := filepath.Join(ea, "The Sims 4", "Mods")
	if got := localizedFolder(p, names); got != p {
		t.Fatalf("with no folder yet = %q, want %q", got, p)
	}
	for _, n := range []string{"The Sims 4", "Die Sims 4"} {
		if err := os.MkdirAll(filepath.Join(ea, n), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(ea, "The Sims 4"), old, old); err != nil {
		t.Fatal(err)
	}
	if got, want := localizedFolder(p, names), filepath.Join(ea, "Die Sims 4", "Mods"); got != want {
		t.Fatalf("= %q, want %q", got, want)
	}
}

// Easy Save 3 writes .bac backups and .tmp files beside a Lethal Company slot; they belong to that slot, not the list.
func TestLethalCompanySideFilesTravelWithTheirSlot(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"LCSaveFile1", "LCSaveFile1.bac", "LCSaveFile1.tmp", "LCSaveFile4", "LCChallengeFile", "LCGeneralSaveData"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	l := SaveLayout("lethal-company", dir)
	names, err := l.Names()
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(names)
	if want := []string{"LCChallengeFile", "LCSaveFile1", "LCSaveFile4"}; !slices.Equal(names, want) {
		t.Fatalf("saves = %v, want %v", names, want)
	}
	if got, want := l.Paths("LCSaveFile1"), []string{"LCSaveFile1", "LCSaveFile1.bac", "LCSaveFile1.tmp"}; !slices.Equal(got, want) {
		t.Fatalf("slot 1 = %v, want %v", got, want)
	}
}

// Only a SMAPI game has a Mods folder in its install that Mortar reads manifest mods from and watches.
func TestManifestModsFolderBelongsToTheSMAPIGameOnly(t *testing.T) {
	if got, ok := ManifestModsFolder("stardew"); !ok || got != "Mods" {
		t.Errorf("stardew = %q, %v", got, ok)
	}
	for _, id := range []string{"lethal-company", "schedule-i", "sims4"} {
		if got, ok := ManifestModsFolder(id); ok {
			t.Errorf("%s = %q, want none", id, got)
		}
	}
}
