package thunderstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// benchLoaderVersions times the first Versions of the loader package against a real cached listing, named by
// MORTAR_BENCH_THUNDERSTORE (a cache folder's thunderstore directory); side keeps the side file between runs.
func benchLoaderVersions(b *testing.B, side bool) {
	src := os.Getenv("MORTAR_BENCH_THUNDERSTORE")
	if src == "" {
		b.Skip("MORTAR_BENCH_THUNDERSTORE names no cached listing")
	}
	cache := b.TempDir()
	dir := filepath.Join(cache, "thunderstore")
	if err := os.CopyFS(dir, os.DirFS(src)); err != nil {
		b.Fatal(err)
	}
	var meta cacheMeta
	raw, err := fsx.ReadFile(filepath.Join(dir, "lethal-company.meta.json"))
	if err != nil || json.Unmarshal(raw, &meta) != nil {
		b.Fatal("no lethal-company.meta.json in the cache")
	}
	d := Driver{CacheDir: cache, Now: func() time.Time { return meta.Fetched.Add(time.Minute) }}
	if _, err := d.Versions(b.Context(), "lethal-company", "BepInEx", "BepInExPack", ""); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		memoMu.Lock()
		clear(memo)
		memoMu.Unlock()
		if !side {
			_ = os.Remove(sidePath(dir, "lethal-company"))
		}
		if _, err := d.Versions(b.Context(), "lethal-company", "BepInEx", "BepInExPack", ""); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoaderVersionsParsingTheListing(b *testing.B) { benchLoaderVersions(b, false) }

func BenchmarkLoaderVersionsFromTheSideFile(b *testing.B) { benchLoaderVersions(b, true) }
