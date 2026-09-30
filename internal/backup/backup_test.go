package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func TestSavesZipsAndKeepsFive(t *testing.T) {
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
		if last, err = Saves(saves, out, start.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	items, _ := os.ReadDir(out)
	if len(items) != Keep {
		t.Fatalf("%d files kept, want %d", len(items), Keep)
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

func TestSavesSkipsMissingFolder(t *testing.T) {
	out := filepath.Join(t.TempDir(), "backups")
	got, err := Saves(filepath.Join(t.TempDir(), "none"), out, time.Now())
	if err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("backups folder created for nothing")
	}
}
