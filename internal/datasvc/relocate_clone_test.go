package datasvc

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// A move keeps the profile files that clone the store as clones; it skips itself where the temp dir cannot clone.
func TestRelocateKeepsStoreClonesShared(t *testing.T) {
	const size = 1 << 20
	root := t.TempDir()
	src, dest, def := filepath.Join(root, "src"), filepath.Join(root, "dest"), filepath.Join(root, "def")
	storeMod := filepath.Join(src, "store", "stardew", "k")
	profMod := filepath.Join(src, "profiles", "stardew", "p", "mods", "k")
	for _, d := range []string{storeMod, filepath.Dir(profMod)} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(storeMod, "a.dll"), bytes.Repeat([]byte("a"), size), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeMod, "b.dll"), bytes.Repeat([]byte("b"), size), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := datadir.MaterializeTreeExclusive(storeMod, profMod); err != nil {
		t.Fatal(err)
	}
	// Same length as the store's b.dll but different bytes: it must be copied, not cloned.
	if err := os.WriteFile(filepath.Join(profMod, "b.dll"), bytes.Repeat([]byte("c"), size), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := Measure(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if before.Total >= 4*size {
		t.Skip("temp dir cannot clone files")
	}
	if err := datadir.Relocate(src, dest, def, extentTotal); err != nil {
		t.Fatal(err)
	}
	after, err := Measure(dest, nil)
	if err != nil {
		t.Fatal(err)
	}
	if after.Total > before.Total+size/4 {
		t.Fatalf("clones were copied: %d bytes on disk after, %d before", after.Total, before.Total)
	}
	got, err := fsx.ReadFile(filepath.Join(dest, "profiles", "stardew", "p", "mods", "k", "b.dll"))
	if err != nil || !bytes.Equal(got, bytes.Repeat([]byte("c"), size)) {
		t.Fatalf("edited file changed: %v", err)
	}
}
