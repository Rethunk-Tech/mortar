package nexus

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestOpenSeenStoreRejectsEmptyDir(t *testing.T) {
	t.Parallel()
	if _, err := OpenSeenStore(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestMarkSeenPersistsAndReloads(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := OpenSeenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSeen("541", 1_700_000_000, "1.2.0"); err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get("541")
	if !ok || got.NewestFileUnix != 1_700_000_000 || got.NewestChange != "1.2.0" || got.LastLookedUnix == 0 {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
	s2, err := OpenSeenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok = s2.Get("541")
	if !ok || got.NewestFileUnix != 1_700_000_000 || got.NewestChange != "1.2.0" {
		t.Fatalf("reloaded %+v ok=%v", got, ok)
	}
	if _, err := os.Stat(filepath.Join(dir, seenFileName)); err != nil {
		t.Fatal(err)
	}
}

func TestMarkSeenEvictsOldestWhenOverCap(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := OpenSeenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := range MaxSeenMods + 1 {
		id := strconv.Itoa(i)
		if err := s.MarkSeen(id, int64(i), id); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(s.Snapshot()); n != MaxSeenMods {
		t.Fatalf("len=%d want %d", n, MaxSeenMods)
	}
}

func TestMarkSeenRejectsEmptyID(t *testing.T) {
	t.Parallel()
	s, err := OpenSeenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSeen("", 1, "1"); err == nil {
		t.Fatal("expected error")
	}
}
