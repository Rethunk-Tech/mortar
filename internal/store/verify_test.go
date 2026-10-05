package store

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func addZip(t *testing.T, s *Store, files map[string]string) string {
	t.Helper()
	key, err := s.AddArchive("stardew", buildZip(t, files))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestVerifyReportsMissingChangedAndExtraFiles(t *testing.T) {
	s := newStore(t)
	key := addZip(t, s, map[string]string{"Mod/a.txt": "aaa", "Mod/b.txt": "bbb", "Mod/c.txt": "ccc"})
	ctx := context.Background()
	if d, err := s.Verify(ctx, "stardew", key); err != nil || !d.Empty() {
		t.Fatalf("fresh item = %+v, %v", d, err)
	}
	dir := filepath.Join(s.root, blobsDir, strings.TrimPrefix(key, "local-"), "Mod")
	if err := os.Remove(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, dir, "b.txt", "BBB")
	testfs.WriteFile(t, dir, "new.txt", "x")
	d, err := s.Verify(ctx, "stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(d.Missing, []string{"Mod/a.txt"}) || !slices.Equal(d.Changed, []string{"Mod/b.txt"}) || !slices.Equal(d.Extra, []string{"Mod/new.txt"}) {
		t.Fatalf("damage = %+v", d)
	}
	if got := s.Damaged("stardew"); len(got) != 1 || got[key].Count() != 3 {
		t.Fatalf("Damaged = %+v", got)
	}
}

func TestVerifyBaselinesAnItemStoredWithoutHashes(t *testing.T) {
	s := newStore(t)
	key := addZip(t, s, map[string]string{"Mod/a.txt": "aaa"})
	if err := os.Remove(s.manifestPath("stardew", key)); err != nil {
		t.Fatal(err)
	}
	if d, err := s.Verify(context.Background(), "stardew", key); err != nil || !d.Empty() {
		t.Fatalf("baseline = %+v, %v", d, err)
	}
	if err := os.WriteFile(filepath.Join(s.root, blobsDir, strings.TrimPrefix(key, "local-"), "Mod", "a.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if d, _ := s.Verify(context.Background(), "stardew", key); len(d.Changed) != 1 {
		t.Fatalf("after baseline = %+v", d)
	}
}

func TestRefsOnlyListsDueItemsAndRemoveForgetsThem(t *testing.T) {
	s := newStore(t)
	key := addZip(t, s, map[string]string{"Mod/a.txt": "aaa"})
	now := time.Now()
	if due, _ := s.Refs(true, now); len(due) != 1 {
		t.Fatalf("never verified is due: %v", due)
	}
	if _, err := s.Verify(context.Background(), "stardew", key); err != nil {
		t.Fatal(err)
	}
	if due, _ := s.Refs(true, now); len(due) != 0 {
		t.Fatalf("just verified is not due: %v", due)
	}
	if due, _ := s.Refs(true, now.Add(VerifyEvery+time.Hour)); len(due) != 1 {
		t.Fatalf("a week later it is due: %v", due)
	}
	if err := s.Remove([]Ref{{Game: "stardew", Key: key}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.manifestPath("stardew", key)); !os.IsNotExist(err) {
		t.Fatalf("hash record survived Remove: %v", err)
	}
}

func TestQuarantineSetsAsideAndRestores(t *testing.T) {
	s := newStore(t)
	key := addZip(t, s, map[string]string{"Mod/a.txt": "aaa"})
	restore, err := s.Quarantine("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Path("stardew", key); err == nil {
		t.Fatal("item still in the store")
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	if d, err := s.Verify(context.Background(), "stardew", key); err != nil || !d.Empty() {
		t.Fatalf("restored = %+v, %v", d, err)
	}
}
