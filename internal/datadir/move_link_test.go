package datadir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRelocateKeepsLinkedProfile(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "data")
	ext := filepath.Join(base, "elsewhere", "p1")
	for _, d := range []string{filepath.Join(src, "profiles", "g"), ext} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(ext, "profile.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LinkDir(filepath.Join(src, "profiles", "g", "p1"), ext); err != nil {
		t.Skip(err)
	}
	dest := filepath.Join(base, "new")
	if err := Relocate(src, dest, filepath.Join(base, "def"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "profiles", "g", "p1", "profile.json")); err != nil {
		t.Fatal(err)
	}
}
