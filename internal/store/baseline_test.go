package store

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestBaselineFromArchiveUnverifiedAndMeta(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	ctx := t.Context()
	if s.HasBaseline("stardew", "local-missing") {
		t.Fatal("HasBaseline before any item")
	}
	key, err := s.AddArchive(ctx, "stardew", buildZip(t, map[string]string{"Mod/a.txt": "aaa"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(s.manifestPath("stardew", key)); err != nil {
		t.Fatal(err)
	}
	if s.HasBaseline("stardew", key) {
		t.Fatal("HasBaseline before first Verify")
	}
	if d, err := s.Verify(ctx, "stardew", key); err != nil || !d.Empty() {
		t.Fatalf("first Verify = %+v, %v", d, err)
	}
	if !s.HasBaseline("stardew", key) {
		t.Fatal("HasBaseline after first Verify")
	}
	if src, pkg, version, ok := s.Meta("stardew", key); !ok || src != "local" || pkg != "" || version != "" {
		t.Fatalf("Meta = %q %q %q %v", src, pkg, version, ok)
	}
	if got := s.Asset("stardew", key); got != "" {
		t.Fatalf("Asset = %q", got)
	}

	other := buildZip(t, map[string]string{"Mod/a.txt": "BBB"})
	if err := s.BaselineFromArchive(ctx, "stardew", key, other); err != nil {
		t.Fatal(err)
	}
	d, err := s.Verify(ctx, "stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(d.Changed, []string{"Mod/a.txt"}) || len(d.Missing) != 0 || len(d.Extra) != 0 {
		t.Fatalf("delta = %+v", d)
	}

	src := t.TempDir()
	testfs.WriteFile(t, src, "b.txt", "backup")
	const backupKey = "nexus-1-1"
	if err := s.AddBackupDir(ctx, "stardew", backupKey, src); err != nil {
		t.Fatal(err)
	}
	if !s.Unverified("stardew", backupKey) {
		t.Fatal("AddBackupDir left Unverified false")
	}
	if err := s.AddArchiveKey(ctx, "stardew", backupKey, buildZip(t, map[string]string{"b.txt": "from-archive"})); err != nil {
		t.Fatal(err)
	}
	if s.Unverified("stardew", backupKey) {
		t.Fatal("AddArchiveKey left Unverified true")
	}
}

// A peer can store a folder under the local key of an archive it does not hold (the 0.3.8 to 0.7.0 LAN shape): the
// key is flagged from its record alone, and the real archive replaces it.
func TestForgedLocalKeyIsUnverifiedAndTheArchiveReplacesIt(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	ctx := t.Context()
	zip := buildZip(t, map[string]string{"Mod/a.txt": "real"})
	key, err := hashKey(zip)
	if err != nil {
		t.Fatal(err)
	}
	forged := t.TempDir()
	testfs.WriteFile(t, forged, "Mod/a.txt", "forged")
	if err := s.AddDir(ctx, "stardew", key, forged); err != nil {
		t.Fatal(err)
	}
	if !s.Unverified("stardew", key) {
		t.Fatal("a local key whose folder is not its archive reads as verified")
	}
	if got, err := s.AddArchive(ctx, "stardew", zip); err != nil || got != key {
		t.Fatalf("AddArchive = %q, %v; want %q", got, err, key)
	}
	if s.Unverified("stardew", key) {
		t.Fatal("the real archive left the key flagged")
	}
	dir, err := s.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := safefs.ReadFile(filepath.Join(dir, "Mod", "a.txt")); err != nil || string(got) != "real" {
		t.Fatalf("content = %q, %v; want the archive's", got, err)
	}
}
