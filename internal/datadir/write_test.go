package datadir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

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
	if b, err := safefs.ReadFile(path); err != nil || string(b) != "x" {
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
