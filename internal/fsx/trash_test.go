//go:build !windows

package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrashMovesIntoTheFreedesktopTrashWithItsOrigin(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	for range 2 {
		dir := filepath.Join(data, "My Mod")
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := Trash(dir); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("%s is still there", dir)
		}
	}
	info, err := ReadFile(filepath.Join(data, "Trash", "info", "My Mod.2.trashinfo"))
	if err != nil || !strings.Contains(string(info), "Path="+filepath.ToSlash(data)+"/My%20Mod\n") {
		t.Fatalf("second trashinfo = %q, %v", info, err)
	}
	if !IsDir(filepath.Join(data, "Trash", "files", "My Mod")) || !IsDir(filepath.Join(data, "Trash", "files", "My Mod.2")) {
		t.Fatal("trashed folders missing")
	}
}
