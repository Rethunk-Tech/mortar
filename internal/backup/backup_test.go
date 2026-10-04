package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestSavesZipsAndKeepsTheChosenCount(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	if err := os.MkdirAll(filepath.Join(saves, "Farm_1"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(saves, "Farm_1", "Farm_1"), []byte("save"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var last string
	for i := range 7 {
		var err error
		if last, err = Saves(saves, out, 3, start.Add(time.Duration(i)*MinGap), Cause{}); err != nil {
			t.Fatal(err)
		}
	}
	items, _ := os.ReadDir(out)
	if len(items) != 3 {
		t.Fatalf("%d files kept, want 3", len(items))
	}
	if _, err := os.Stat(filepath.Join(out, "2026-01-01T00-00-00.000.zip")); err == nil {
		t.Fatal("oldest backup survived")
	}
	zr, err := zip.OpenReader(last)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	if len(zr.File) != 1 || zr.File[0].Name != "Saves/Farm_1/Farm_1" {
		t.Fatalf("zip holds %v", zr.File)
	}
}

func TestPinnedBackupSurvivesRotation(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	if err := os.MkdirAll(filepath.Join(saves, "Farm_1"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(saves, "Farm_1", "Farm_1"), []byte("save"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first, err := Saves(saves, out, 2, start, Cause{Kind: KindLaunch})
	if err != nil {
		t.Fatal(err)
	}
	if err := SetPinned(out, filepath.Base(first), true); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < 5; i++ {
		if _, err := Saves(saves, out, 2, start.Add(time.Duration(i)*MinGap), Cause{Kind: KindLaunch}); err != nil {
			t.Fatal(err)
		}
	}
	items, err := List(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("%d backups, want pinned plus two unpinned", len(items))
	}
	if !items[len(items)-1].Pinned {
		t.Fatalf("pinned backup missing: %+v", items)
	}
}

func TestSavesSkipsMissingFolder(t *testing.T) {
	out := filepath.Join(t.TempDir(), "backups")
	got, err := Saves(filepath.Join(t.TempDir(), "none"), out, DefaultKeep, time.Now(), Cause{})
	if err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("backups folder created for nothing")
	}
}

func TestSavesSkipsWhileTheNewestIsRecent(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	if err := os.MkdirAll(saves, 0o750); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(saves, start.Add(-time.Hour), start.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	first, err := Saves(saves, out, DefaultKeep, start, Cause{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < 8; i++ {
		got, err := Saves(saves, out, DefaultKeep, start.Add(time.Duration(i)*time.Minute), Cause{})
		if err != nil || got != first {
			t.Fatalf("backup %d = %q, %v; want the existing %q", i, got, err, first)
		}
	}
	if _, err := Saves(saves, out, DefaultKeep, start.Add(MinGap), Cause{}); err != nil {
		t.Fatal(err)
	}
	if items, _ := os.ReadDir(out); len(items) != 2 {
		t.Fatalf("%d backups, want 2", len(items))
	}
}

func TestSavesBacksUpAgainWhenASaveChangedAndSweepsCrashedTemps(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	if err := os.MkdirAll(saves, 0o750); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(saves, start.Add(-time.Hour), start.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	crashed := filepath.Join(out, "backup-1.tmp")
	if err := fsx.WriteFile(crashed, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(crashed, start.Add(-time.Hour), start.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	first, err := Saves(saves, out, DefaultKeep, start, Cause{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(crashed); err == nil {
		t.Fatal("a crashed temp file survived")
	}
	farm := filepath.Join(saves, "Farm_1")
	if err := fsx.WriteFile(farm, []byte("day 2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(farm, start.Add(time.Minute), start.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := Saves(saves, out, DefaultKeep, start.Add(2*time.Minute), Cause{})
	if err != nil || got == first {
		t.Fatalf("backup after a save changed = %q, %v; want a new one", got, err)
	}
}

func TestFolderZipsOnlyThatSaveAndStaysPinned(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	for _, folder := range []string{"Farm_1", "Farm_2"} {
		if err := os.MkdirAll(filepath.Join(saves, folder), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(filepath.Join(saves, folder, folder), []byte(folder), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	out := t.TempDir()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if _, err := Saves(saves, out, 1, start, Cause{Kind: KindLaunch}); err != nil {
		t.Fatal(err)
	}
	got, err := Folder(saves, out, "Farm_1", 1, start.Add(MinGap), Cause{Kind: KindManual, Pinned: true})
	if err != nil {
		t.Fatal(err)
	}
	items, err := List(out)
	if err != nil {
		t.Fatal(err)
	}
	var manual Backup
	for _, b := range items {
		if b.Name == filepath.Base(got) {
			manual = b
		}
	}
	if manual.Kind != KindManual || !manual.Pinned || len(manual.Saves) != 1 || manual.Saves[0].Folder != "Farm_1" {
		t.Fatalf("manual backup = %+v", manual)
	}
	zr, err := zip.OpenReader(got)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	if len(zr.File) != 1 || zr.File[0].Name != "Saves/Farm_1/Farm_1" {
		t.Fatalf("zip holds %v", zr.File)
	}
	if _, err := Saves(saves, out, 1, start.Add(2*MinGap), Cause{Kind: KindLaunch}); err != nil {
		t.Fatal(err)
	}
	after, err := List(out)
	if err != nil {
		t.Fatal(err)
	}
	kept := false
	for _, b := range after {
		if b.Name == filepath.Base(got) && b.Pinned {
			kept = true
		}
	}
	if !kept {
		t.Fatalf("manual backup pruned: %+v", after)
	}
}

func TestABackwardsClockKeepsTheNewZip(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	if err := os.MkdirAll(saves, 0o750); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := Saves(saves, out, 1, start.Add(time.Hour), Cause{}); err != nil {
		t.Fatal(err)
	}
	got, err := Saves(saves, out, 1, start, Cause{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("the zip just written is gone: %v", err)
	}
}

func TestSaveDirRejectsEscapesAndMissing(t *testing.T) {
	saves := t.TempDir()
	if err := os.Mkdir(filepath.Join(saves, "Farm_1"), 0o750); err != nil {
		t.Fatal(err)
	}
	if dir, err := SaveDir(saves, "Farm_1"); err != nil || dir != filepath.Join(saves, "Farm_1") {
		t.Fatalf("SaveDir = %q, %v", dir, err)
	}
	for _, folder := range []string{"", ".", "..", "../Farm_1", "Gone"} {
		if _, err := SaveDir(saves, folder); err == nil {
			t.Fatalf("SaveDir(%q) accepted", folder)
		}
	}
}

func TestKindsRotateSeparately(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	if err := os.MkdirAll(filepath.Join(saves, "A_1"), 0o700); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	start := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	for i, kind := range []string{KindLaunch, KindUpdate, KindLaunch, KindUpdate} {
		if _, err := Saves(saves, out, 1, start.Add(time.Duration(i)*time.Hour), Cause{Kind: kind}); err != nil {
			t.Fatal(err)
		}
	}
	zips, err := list(out)
	if err != nil || len(zips) != 2 {
		t.Fatalf("zips = %v, %v: each kind keeps its own newest", zips, err)
	}
}
