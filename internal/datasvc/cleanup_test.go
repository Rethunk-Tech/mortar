package datasvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestSelectKeepsReferencedAndApplyRemovesTheRest(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	put := func(rel string, body []byte) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "store", "stardew", "keep", "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "store", "stardew", "keep", "x", "a.bin"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "store", "stardew", "gone"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "store", "stardew", "gone", "b.bin"), []byte("gone!"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	old, _ := json.Marshal(fetchedFile{Fetched: now.Add(-48 * time.Hour)})
	fresh, _ := json.Marshal(fetchedFile{Fetched: now.Add(-time.Hour)})
	put("cache/nexus/details-v3-stardewvalley-1.json", old)
	put("cache/nexus/details-v3-stardewvalley-2.json", fresh)
	if err := os.MkdirAll(filepath.Join(root, "cache", ".tmp-empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cache", ".tmp_legacy"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cache", ".tmp_legacy", "partial"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cache", "write.tmp"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := Select(root, items, map[string][]string{"stardew": {"keep"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, it := range preview.Items {
		kinds[it.Kind]++
		if it.Rel == "store/stardew/keep" {
			t.Fatalf("referenced store item listed: %+v", preview.Items)
		}
	}
	if kinds["store"] != 1 || kinds["cache"] != 1 || kinds["temp"] != 3 {
		t.Fatalf("kinds = %v items = %+v", kinds, preview.Items)
	}
	if err := Apply(root, items, preview, map[string][]string{"stardew": {"keep"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "store", "stardew", "keep", "x", "a.bin")); err != nil {
		t.Fatal("referenced item was removed")
	}
	if _, err := os.Stat(filepath.Join(root, "store", "stardew", "gone")); !os.IsNotExist(err) {
		t.Fatalf("unused store item remains: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "cache", "nexus", "details-v3-stardewvalley-1.json")); !os.IsNotExist(err) {
		t.Fatal("expired cache remains")
	}
	if _, err := os.Stat(filepath.Join(root, "cache", "nexus", "details-v3-stardewvalley-2.json")); err != nil {
		t.Fatal("fresh cache was removed")
	}
	if _, err := os.Stat(filepath.Join(root, "cache", ".tmp-empty")); !os.IsNotExist(err) {
		t.Fatal("empty temp remains")
	}
	if _, err := os.Stat(filepath.Join(root, "cache", ".tmp_legacy")); !os.IsNotExist(err) {
		t.Fatal("underscore temp remains")
	}
	if _, err := os.Stat(filepath.Join(root, "cache", "write.tmp")); !os.IsNotExist(err) {
		t.Fatal("tmp file remains")
	}
}

func TestApplySkipsAStoreKeyThatBecameReferenced(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(root, "store", "stardew", "gone")
	if err := os.MkdirAll(gone, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gone, "b.bin"), []byte("gone!"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	preview, err := Select(root, items, map[string][]string{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(root, items, preview, map[string][]string{"stardew": {"gone"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(gone, "b.bin")); err != nil {
		t.Fatal("store item referenced after preview was removed")
	}
}
