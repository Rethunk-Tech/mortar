package steam

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestBackupBeforeEditOnlyCopiesOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "steam.vdf")
	if err := fsx.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := backupBeforeEdit(path, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteFile(path, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := backupBeforeEdit(path, []byte("newer"), 0o600); err != nil {
		t.Fatal(err)
	}
	body, err := fsx.ReadFile(filepath.Join(dir, "steam.vdf.mortar.bak"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "old" {
		t.Fatalf("backup = %q", body)
	}
}

func TestAtomicWriteFileLeavesOldFileOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "steam.vdf")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteFile(filepath.Join(dir, "missing", "steam.vdf"), []byte("new"), 0o600); err == nil {
		t.Fatal("expected write failure")
	}
	body, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "old" {
		t.Fatalf("old file changed: %q", body)
	}
}
