package datadir

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func TestRelocateCopiesThenRemovesAndLeavesAPointer(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	src := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "settings.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "new")
	def := t.TempDir()
	if err := Relocate(src, dest, def); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(filepath.Join(dest, "settings.json"))
	if err != nil || string(got) != "{}" {
		t.Fatalf("copied = %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(src, "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("old copy remains: %v", err)
	}
	ptr, err := fsx.ReadFile(filepath.Join(def, PointerName))
	if err != nil || !strings.Contains(string(ptr), dest) {
		t.Fatalf("pointer = %q %v", ptr, err)
	}
}

func TestRelocateRefusesATargetInsideTheSourceOrANonEmptyFolder(t *testing.T) {
	src := t.TempDir()
	if err := Relocate(src, filepath.Join(src, "inside"), t.TempDir()); !errors.Is(err, ErrInside) {
		t.Fatalf("inside = %v", err)
	}
	dest := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(dest, "x"), []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Relocate(src, dest, t.TempDir()); !errors.Is(err, ErrNotEmpty) {
		t.Fatalf("empty = %v", err)
	}
}

func TestRelocateRemovesPartialDestinationWhenCopyFails(t *testing.T) {
	src := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "a"), []byte("copied"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), filepath.Join(src, "z")); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "new")
	err := Relocate(src, dest, t.TempDir())
	if err == nil {
		t.Fatal("copy unexpectedly succeeded")
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("partial destination remains: %v", statErr)
	}
}

func TestResolveFollowsThePointerFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", home)
	t.Setenv("LOCALAPPDATA", home)
	dest := t.TempDir()
	def, err := defaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(def, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(filepath.Join(def, PointerName), []byte(dest+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := resolve()
	if err != nil || got != dest {
		t.Fatalf("resolve = %q %v", got, err)
	}
}
