package scratch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestSweepRemovesOnlyOldScratchFolders(t *testing.T) {
	testfs.DataHome(t)
	old, err := Dir("mortar-old-")
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := Dir("mortar-fresh-")
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	n, err := Sweep(24 * time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("Sweep = %d, %v; want 1", n, err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old folder kept")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("fresh folder removed")
	}
}

func TestDirInMakesTheFolderUnderTheGivenBasesCache(t *testing.T) {
	base := t.TempDir()
	dir, err := DirIn(base, "mortar-x-")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(dir) != filepath.Join(base, "cache") || !strings.HasPrefix(filepath.Base(dir), "mortar-x-") {
		t.Fatalf("dir = %q", dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("stat = %v, %v", info, err)
	}
}
