package datadir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileRemovesTempOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("data"), 0o600); err == nil {
		t.Fatal("expected rename failure onto a directory")
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if e.Name() != "target" {
			t.Fatalf("leftover %q", e.Name())
		}
	}
}
