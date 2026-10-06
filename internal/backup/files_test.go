package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/saves"
)

// lethalCompany is a Lethal Company save folder: two slots and the challenge file beside files that are not saves.
func lethalCompany(t *testing.T) saves.Layout {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "Lethal Company")
	for _, name := range []string{"LCSaveFile1", "LCSaveFile2", "LCChallengeFile", "LCGeneralSaveData", "Player.log", "InputUtils/binds.json"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(filepath.Join(dir, name), []byte("v1 "+name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return saves.Layout{Dir: dir, Files: []string{"LCSaveFile*", "LCChallengeFile"}}
}

func zipNames(t *testing.T, p string) []string {
	t.Helper()
	zr, err := zip.OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var out []string
	for _, f := range zr.File {
		out = append(out, f.Name)
	}
	slices.Sort(out)
	return out
}

func TestFileSavesBackUpAndRestore(t *testing.T) {
	l := lethalCompany(t)
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	whole, err := Saves(l, out, DefaultKeep, start, Cause{Kind: KindUpdate})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := zipNames(t, whole), []string{"Saves/LCChallengeFile", "Saves/LCSaveFile1", "Saves/LCSaveFile2"}; !slices.Equal(got, want) {
		t.Fatalf("whole zip = %v, want %v", got, want)
	}
	listed, err := List(out, l)
	if err != nil || len(listed) != 1 || len(listed[0].Saves) != 3 || listed[0].Saves[1] != (Snap{Folder: "LCSaveFile1"}) {
		t.Fatalf("List = %+v, %v; want a save file listed by its folder with no farm name", listed, err)
	}
	one, err := Folder(l, out, "LCSaveFile2", DefaultKeep, start.Add(time.Minute), Cause{Kind: KindManual, Pinned: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := zipNames(t, one); !slices.Equal(got, []string{"Saves/LCSaveFile2"}) {
		t.Fatalf("one-save zip = %v", got)
	}
	if run, err := Scheduled(l, out, DefaultKeep, start.Add(time.Hour)); err != nil || run.Saved != 3 {
		t.Fatalf("Scheduled = %+v, %v", run, err)
	}

	for _, name := range []string{"LCSaveFile1", "LCSaveFile2"} {
		if err := fsx.WriteFile(filepath.Join(l.Dir, name), []byte("v2"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := Restore(whole, l, out, []string{"LCSaveFile1"}, DefaultKeep, start.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"LCSaveFile1": "v1 LCSaveFile1", "LCSaveFile2": "v2", "Player.log": "v1 Player.log"} {
		if b, err := fsx.ReadFile(filepath.Join(l.Dir, name)); err != nil || string(b) != want {
			t.Errorf("%s = %q, %v; want %q", name, b, err, want)
		}
	}
	if err := Restore(whole, l, out, []string{"LCGeneralSaveData"}, DefaultKeep, start.Add(3*time.Hour)); err == nil {
		t.Fatal("restored a file that is not a save")
	}
}

// TestFileSetSavesInFoldersBackUpAndRestore is Valheim's layout: characters and worlds in their own folders, a world
// being its .fwl with the .db sharing its stem, beside .old copies and timestamped backups that are not saves.
func TestFileSetSavesInFoldersBackUpAndRestore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Valheim")
	for _, name := range []string{
		"characters_local/Ragnar.fch", "characters_local/Ragnar.fch.old",
		"worlds_local/Midgard.fwl", "worlds_local/Midgard.db", "worlds_local/Midgard.fwl.old", "worlds_local/Midgard.db.old",
		"worlds_local/Midgard_backup_auto-20260101120000.fwl", "worlds_local/Midgard_backup_auto-20260101120000.db",
		"Player.log",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(filepath.Join(dir, name), []byte("v1 "+name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	l := saves.Layout{Dir: dir, Files: []string{"characters_local/*.fch", "worlds_local/*.fwl", "!*_backup_*"}, Companions: []string{".db"}}
	if names, err := l.Names(); err != nil || !slices.Equal(names, []string{"characters_local/Ragnar.fch", "worlds_local/Midgard.fwl"}) {
		t.Fatalf("Names = %v, %v", names, err)
	}
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	whole, err := Saves(l, out, DefaultKeep, start, Cause{Kind: KindUpdate})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := zipNames(t, whole), []string{"Saves/characters_local/Ragnar.fch", "Saves/worlds_local/Midgard.db", "Saves/worlds_local/Midgard.fwl"}; !slices.Equal(got, want) {
		t.Fatalf("whole zip = %v, want %v", got, want)
	}
	listed, err := List(out, l)
	if err != nil || len(listed) != 1 || !slices.Equal(listed[0].Saves, []Snap{{Folder: "characters_local/Ragnar.fch"}, {Folder: "worlds_local/Midgard.fwl"}}) {
		t.Fatalf("List = %+v, %v", listed, err)
	}
	for _, name := range []string{"worlds_local/Midgard.fwl", "worlds_local/Midgard.db"} {
		if err := fsx.WriteFile(filepath.Join(dir, name), []byte("v2"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := Restore(whole, l, out, []string{"worlds_local/Midgard.fwl"}, DefaultKeep, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"worlds_local/Midgard.fwl", "worlds_local/Midgard.db", "worlds_local/Midgard.db.old"} {
		if b, err := fsx.ReadFile(filepath.Join(dir, name)); err != nil || string(b) != "v1 "+name {
			t.Errorf("%s = %q, %v", name, b, err)
		}
	}
}
