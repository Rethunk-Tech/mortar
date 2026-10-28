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
	for i := range MaxSeenMods {
		s.data.Mods[strconv.Itoa(i)] = SeenEntry{LastLookedUnix: int64(i + 1)}
	}
	if err := s.MarkSeen("new", 1, "1"); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if len(snap) != MaxSeenMods {
		t.Fatalf("len=%d want %d", len(snap), MaxSeenMods)
	}
	if _, ok := snap["0"]; ok {
		t.Fatal("oldest entry not evicted")
	}
	if _, ok := snap["new"]; !ok {
		t.Fatal("newest entry evicted")
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

func TestOpenSeenStoreMovesCorruptFileAside(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, seenFileName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenSeenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot()) != 0 {
		t.Fatal("expected empty store")
	}
	if m, _ := filepath.Glob(filepath.Join(dir, seenFileName+".corrupt-*")); len(m) != 1 {
		t.Fatalf("corrupt copy: %v", m)
	}
}
