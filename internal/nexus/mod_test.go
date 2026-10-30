package nexus

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
)

var stardew = nexusmods.Title{Domain: "stardewvalley", ID: 1303}

func modServer(t *testing.T, hits *atomic.Int32) *nexusmods.Client {
	t.Helper()
	body, err := os.ReadFile("testdata/mod-541.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	c := nexusmods.New("Mortar", "9.9")
	c.BaseURL = srv.URL
	return c.WithKey("k")
}

var wantMod = Mod{Name: "Lookup Anything", Author: "Pathoschild", PictureURL: "https://staticdelivery.nexusmods.com/mods/1303/images/541-0-1482010183.png", EndorsementCount: 201516, Summary: "See live info about whatever's under your cursor when you press F1. Learn a villager's favourite gifts, when a crop will be ready to harvest, how long a fence will last, why your farm animals are unhappy, and more."}

func TestCachedModIsFetchedOnce(t *testing.T) {
	var hits atomic.Int32
	c := modServer(t, &hits)
	cache := &diskcache.Cache{Dir: t.TempDir()}
	for range 2 {
		m, err := CachedMod(t.Context(), c, cache, stardew, 541)
		if err != nil || m != wantMod {
			t.Fatalf("mod = %+v, %v", m, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d, want the second call served from disk", hits.Load())
	}
	if _, err := os.Stat(filepath.Join(cache.Dir, "nexus", "mod-v2-stardewvalley-541.json")); err != nil {
		t.Fatalf("the entry is not under the versioned name: %v", err)
	}
}

// A bare Mod under the old name, or a value-less file under the new one, must not answer as an empty Mod forever.
func TestCachedModIgnoresEarlierCacheShapes(t *testing.T) {
	var hits atomic.Int32
	c := modServer(t, &hits)
	cache := &diskcache.Cache{Dir: t.TempDir()}
	if err := os.MkdirAll(filepath.Join(cache.Dir, "nexus"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"nexus/mod-stardewvalley-541.json":    `{"name":"stale"}`,
		"nexus/mod-v2-stardewvalley-541.json": `{"fetched":"2026-09-30T12:00:00Z","name":"stale"}`,
	} {
		if err := os.WriteFile(filepath.Join(cache.Dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	m, err := CachedMod(t.Context(), c, cache, stardew, 541)
	if err != nil || m != wantMod || hits.Load() != 1 {
		t.Fatalf("mod = %+v, hits %d, %v", m, hits.Load(), err)
	}
}

func TestCachedModWithoutACacheFetchesEveryTime(t *testing.T) {
	var hits atomic.Int32
	c := modServer(t, &hits)
	for range 2 {
		if _, err := CachedMod(t.Context(), c, nil, stardew, 541); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 2 {
		t.Fatalf("hits = %d, want a fetch per call", hits.Load())
	}
}
