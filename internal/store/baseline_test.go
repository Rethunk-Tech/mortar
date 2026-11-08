package store

import (
	"os"
	"slices"
	"testing"

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
