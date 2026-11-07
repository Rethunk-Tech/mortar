package store

import (
	"os"
	"testing"
)

func TestSavedIndexStampIsTheWrittenFile(t *testing.T) {
	s := newStore(t)
	if err := s.bind("stardew", "local-a", "blob1", false); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(s.indexPath())
	if err != nil {
		t.Fatal(err)
	}
	if s.idxFile == nil || !os.SameFile(fi, s.idxFile) || !fi.ModTime().Equal(s.idxFile.ModTime()) {
		t.Fatal("memo stamp is not the identity of the index file just written")
	}
}

func TestMutationKeepsIndexMemo(t *testing.T) {
	s := newStore(t)
	if err := s.bind("stardew", "local-a", "blob1", false); err != nil {
		t.Fatal(err)
	}
	before := s.idxDecodes.Load()
	if err := s.bind("stardew", "local-b", "blob2", false); err != nil {
		t.Fatal(err)
	}
	idx, err := s.viewIndex()
	if err != nil {
		t.Fatal(err)
	}
	if got := s.idxDecodes.Load() - before; got != 0 {
		t.Fatalf("a mutation and a read decoded the index %d times, want 0", got)
	}
	if _, ok := idx["stardew"]["local-b"]; !ok {
		t.Fatal("memoised index lacks the new key")
	}
}

func TestOtherStoreWriteInvalidatesMemo(t *testing.T) {
	a := newStore(t)
	b := &Store{root: a.root}
	if err := a.bind("stardew", "local-a", "blob1", false); err != nil {
		t.Fatal(err)
	}
	if err := b.bind("stardew", "local-b", "blob2", false); err != nil {
		t.Fatal(err)
	}
	idx, err := a.viewIndex()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx["stardew"]["local-b"]; !ok {
		t.Fatalf("first instance served a stale index: %v", idx)
	}
}
