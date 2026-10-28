package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkFilesGivesSlashPathsAndSizesOfRegularFilesOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "b", "f.txt"), []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	if err := WalkFiles(root, func(rel, _ string, size int64) error { got[rel] = size; return nil }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["a/b/f.txt"] != 3 {
		t.Fatalf("WalkFiles = %v", got)
	}
}
