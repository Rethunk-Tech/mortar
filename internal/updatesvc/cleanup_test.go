package updatesvc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveOldExecutables(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "mortar.exe")
	keep := []string{"mortar.exe", "mortar.exe.old", "mortar.exe.old.x", "other.exe.old.1", "mortar.exe.old.1.bak"}
	for _, n := range append([]string{"mortar.exe.old.1", "mortar.exe.old.23"}, keep...) {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	RemoveOldExecutables(exe)
	left, _ := os.ReadDir(dir)
	if len(left) != len(keep) {
		t.Fatalf("left %d files, want %d", len(left), len(keep))
	}
	for _, n := range keep {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s removed", n)
		}
	}
}
