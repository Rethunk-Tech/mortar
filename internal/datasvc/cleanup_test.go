package datasvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestSelectKeepsReferencedAndApplyRemovesTheRest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	items := store.OpenAt(filepath.Join(root, "store"))
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
	addItem(t, items, "keep", map[string]string{"x/a.bin": "keep"})
	addItem(t, items, "gone", map[string]string{"b.bin": "gone!"})
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
	preview, err := Select(root, items, map[string][]string{"stardew": {"keep"}}, now, nil)
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
	if _, err := items.Dir("stardew", "keep"); err != nil {
		t.Fatal("referenced item was removed")
	}
	if _, err := items.Dir("stardew", "gone"); err == nil {
		t.Fatal("unused store item remains")
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
	t.Parallel()
	root := t.TempDir()
	items := store.OpenAt(filepath.Join(root, "store"))
	addItem(t, items, "gone", map[string]string{"b.bin": "gone!"})
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	preview, err := Select(root, items, map[string][]string{}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(root, items, preview, map[string][]string{"stardew": {"gone"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := items.Dir("stardew", "gone"); err != nil {
		t.Fatal("store item referenced after preview was removed")
	}
}

func TestSelectDropsRetiredCacheAndExpiresNexusPages(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	items := store.OpenAt(filepath.Join(root, "store"))
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	old, _ := json.Marshal(fetchedFile{Fetched: now.Add(-48 * time.Hour)})
	fresh, _ := json.Marshal(fetchedFile{Fetched: now.Add(-time.Hour)})
	files := map[string][]byte{
		"cache/nexus-requirements-stardewvalley-1.json":   fresh,
		"cache/nexus/page-v1-stardewvalley-1.json":        fresh,
		"cache/nexus/page-v2-stardewvalley-1.json":        old,
		"cache/nexus/page-absent-v1-stardewvalley-2.json": old,
		"cache/nexus/page-v2-stardewvalley-3.json":        fresh,
		"cache/smapi-compat.json":                         old,
	}
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	preview, err := Select(root, items, nil, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, it := range preview.Items {
		got[it.Rel] = true
	}
	want := []string{
		"cache/nexus-requirements-stardewvalley-1.json", "cache/nexus/page-v1-stardewvalley-1.json",
		"cache/nexus/page-v2-stardewvalley-1.json", "cache/nexus/page-absent-v1-stardewvalley-2.json",
	}
	if len(got) != len(want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
	for _, rel := range want {
		if !got[rel] {
			t.Fatalf("%s not listed: %v", rel, got)
		}
	}
}

func TestCacheTTLClaimsNexusChangelogsAndCollections(t *testing.T) {
	for _, rel := range []string{"nexus/changelogs-stardewvalley-541.json", "nexus-collection-stardewvalley-abc-3.json"} {
		if ttl, ok := cacheTTL(rel); !ok || ttl <= 0 {
			t.Errorf("cacheTTL(%q) = %v, %v; want an expiry", rel, ttl, ok)
		}
	}
}
