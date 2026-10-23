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

func TestSizeCountsALinkAcrossFoldersOnce(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "m0", "f")
	if err := os.MkdirAll(filepath.Dir(first), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, make([]byte, 50), 0o600); err != nil {
		t.Fatal(err)
	}
	const folders = sizeWalkers * 4
	for i := 1; i < folders; i++ {
		link := filepath.Join(dir, "m"+string(rune('a'+i)), "f")
		if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(first, link); err != nil {
			t.Skipf("no hard links: %v", err)
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(link), "own"), make([]byte, 1), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want := int64(50 + folders - 1)
	if got, err := Size(dir); err != nil || got != want {
		t.Fatalf("Size = %d, %v; want %d", got, err, want)
	}
}
