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
	if err := WriteFileAtomic(path, []byte("data"), 0o600); err == nil {
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
	syncWrites = true
	t.Cleanup(func() { syncWrites = false })
	dir := t.TempDir()
	path := filepath.Join(dir, "target")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteStreamAtomic(path, 0o600, func(w io.Writer) error {
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
	if err := WriteStreamAtomic(path, 0o600, func(io.Writer) error {
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

func TestReadJournalSkipsMissingAndBusyAndReadsTheRest(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "journal")
	var v struct{ N int }
	if found, err := ReadJournal(filepath.Join(dir, "j.json"), nil, &v); found || err != nil {
		t.Fatalf("missing: %v, %v", found, err)
	}
	if err := WriteJSONIn(dir, "j.json", struct{ N int }{7}); err != nil {
		t.Fatal(err)
	}
	if found, err := ReadJournal(filepath.Join(dir, "j.json"), func() bool { return true }, &v); found || err != nil || v.N != 0 {
		t.Fatalf("busy: %v, %v, %d", found, err, v.N)
	}
	if found, err := ReadJournal(filepath.Join(dir, "j.json"), func() bool { return false }, &v); !found || err != nil || v.N != 7 {
		t.Fatalf("read: %v, %v, %d", found, err, v.N)
	}
}

func TestWriteOrRemoveWritesThenRemoves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "f")
	if err := WriteOrRemove(path, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if b, err := fsx.ReadFile(path); err != nil || string(b) != "x" {
		t.Fatalf("written = %q, %v", b, err)
	}
	if err := WriteOrRemove(path, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("not removed: %v", err)
	}
	if err := WriteOrRemove(path, nil); err != nil {
		t.Fatalf("removing a missing file: %v", err)
	}
}
