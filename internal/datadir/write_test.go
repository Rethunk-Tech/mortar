package datadir

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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

func TestWriteStreamReplacesExistingAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteStream(path, 0o600, func(w io.Writer) error {
		_, err := w.Write([]byte("new-content"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(path)
	if err != nil || string(got) != "new-content" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestWriteStreamErrorLeavesNoTempAndKeepsTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteStream(path, 0o600, func(io.Writer) error {
		return errors.New("write failed")
	}); err == nil {
		t.Fatal("expected error")
	}
	got, err := fsx.ReadFile(path)
	if err != nil || string(got) != "keep" {
		t.Fatalf("got %q %v", got, err)
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
