package datadir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSize(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, n := range map[string]int{"x": 3, "a/y": 10, "a/b/z": 100} {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, n), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := Size(dir); err != nil || got != 113 {
		t.Fatalf("Size = %d, %v; want 113", got, err)
	}
	if _, err := Size(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("a missing folder should fail")
	}
}
