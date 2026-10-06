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
	listed, err := List(out)
	if err != nil || len(listed) != 1 || len(listed[0].Saves) != 3 || listed[0].Saves[1].Folder != "LCSaveFile1" {
		t.Fatalf("List = %+v, %v", listed, err)
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
