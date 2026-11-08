package deploy

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// A kept journal moves with the data folder between Plays; DiscardKept is what takes it back from where it now lives.
func TestDiscardKeptReturnsThePlayersFilesAndDropsAJournalThatMoved(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	before := testfs.Tree(t, r.install)
	if err := r.play(t, Mode{Reuse: ReuseStat}); err != nil {
		t.Fatal(err)
	}
	if !Kept(r.view.JournalDir) {
		t.Fatal("a play that leaves its files has no kept journal")
	}
	moved := r.view.JournalDir + "-moved"
	if err := os.Rename(r.view.JournalDir, moved); err != nil {
		t.Fatal(err)
	}
	if Kept(r.view.JournalDir) {
		t.Fatal("a folder with no journal reads as kept")
	}
	if err := (Place{}).DiscardKept(t.Context(), moved); err != nil {
		t.Fatal(err)
	}
	after := testfs.Tree(t, r.install)
	if len(after) != len(before) {
		t.Fatalf("tree differs\nbefore %v\n after %v", before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
	if _, err := os.Stat(moved); !os.IsNotExist(err) {
		t.Fatalf("the kept journal is still there: %v", err)
	}
}

func TestDiscardKeptLeavesAJournalThatDidNotKeepItsFilesAndAnAbsentOne(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.apply()
	if err := (Place{}).DiscardKept(t.Context(), r.view.JournalDir); err != nil {
		t.Fatal(err)
	}
	if !HasJournal(r.view.JournalDir) {
		t.Fatal("a journal that did not keep its files was taken back instead of recovered")
	}
	if got := read(filepath.Join(r.install, "winhttp.dll")); got != "proxy" {
		t.Fatalf("winhttp.dll = %q, want the placed copy left alone", got)
	}
	if err := (Place{}).DiscardKept(t.Context(), filepath.Join(r.root, "no-journal")); err != nil {
		t.Fatalf("a folder with no journal: %v", err)
	}
}

// move falls back to a copy and a remove when the rename cannot cross file systems, which only a second mount shows.
func TestMoveAcrossFileSystemsCopiesThenRemoves(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "winhttp.dll")
	write(t, src, "the player's own")
	shm, err := safefs.Device("/dev/shm")
	if err != nil {
		t.Skipf("no second file system to move across: %v", err)
	}
	if here, err := safefs.Device(srcDir); err != nil || here == shm {
		t.Skip("both folders are on one file system")
	}
	other := filepath.Join("/dev/shm", filepath.Base(srcDir))
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Skipf("cannot make a folder on the second file system: %v", err)
	}
	t.Cleanup(func() { _ = safefs.RemoveAll(other) })
	dst := filepath.Join(other, "displaced", "winhttp.dll")
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := move(src, dst); err != nil {
		t.Fatal(err)
	}
	if got := read(dst); got != "the player's own" {
		t.Fatalf("moved file = %q", got)
	}
	if _, err := os.Lstat(src); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the source is still there: %v", err)
	}
}
