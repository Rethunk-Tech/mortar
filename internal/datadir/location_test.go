package datadir

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestDirRefusesAMissingRelocationTarget(t *testing.T) {
	def := t.TempDir()
	t.Setenv("XDG_DATA_HOME", def)
	t.Setenv("LOCALAPPDATA", def)
	old := portable
	portable = func() (string, error) { return "", nil }
	t.Cleanup(func() { portable = old })
	d, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(t.TempDir(), "unplugged")
	testfs.WriteFile(t, d, PointerName, gone)
	var miss *MissingLocationError
	if _, err := Dir(); !errors.As(err, &miss) {
		t.Fatalf("Dir = %v, want MissingLocationError", err)
	}
	if _, err := os.Stat(gone); err == nil {
		t.Fatal("Dir created the missing target")
	}
	if err := UseDefaultLocation(); err != nil {
		t.Fatal(err)
	}
	if got, err := Dir(); err != nil || got != d {
		t.Fatalf("after UseDefaultLocation Dir = %q, %v", got, err)
	}
	if err := SetLocation(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := Dir(); err != nil {
		t.Fatal(err)
	}
}
