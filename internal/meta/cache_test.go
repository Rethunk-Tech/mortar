package meta

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/go-smapiweb"
)

func TestRetiredCacheNames(t *testing.T) {
	t.Parallel()
	for rel, want := range map[string]bool{
		"github-o-r-0123456789ab-releases.json":    true,
		"github-o-r-v1-0123456789ab-notes.json":    true,
		"nexus/mod-stardewvalley-541.json":         true,
		"nexus-requirements-stardewvalley-1.json":  true,
		"github-o-r-0123456789ab-releases-v2.json": false,
		"github-o-r-v1-0123456789ab-notes-v2.json": false,
		"nexus/mod-v2-stardewvalley-541.json":      false,
		"smapi-release.json":                       false,
	} {
		if got := Retired(rel); got != want {
			t.Errorf("Retired(%q) = %v, want %v", rel, got, want)
		}
	}
}

// An empty CacheDir means <datadir>/cache, so a client built without one still caches; diskcache itself would
// treat an empty Dir as caching switched off.
func TestEmptyCacheDirCachesUnderTheDataFolder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"Id":7,"Name":"Seven"}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), PageBase: srv.URL}
	if got := c.Cache().Dir; !strings.HasPrefix(got, dir) || filepath.Base(got) != "cache" {
		t.Fatalf("cache dir %q, want a cache folder under %q", got, dir)
	}
	for range 2 {
		if _, err := c.Page(context.Background(), 7); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("fetched %d times, want once from the cache", hits.Load())
	}
}

func TestSmapiOutcomeRecordsOneRealFailureOrNone(t *testing.T) {
	first, second := errors.New("batch 0"), errors.New("batch 1")
	joined := errors.Join(first, second)
	known := []smapiweb.UpdateResult{{ID: "A", Known: true}, {ID: "B"}}
	unknown := []smapiweb.UpdateResult{{ID: "A"}, {ID: "B"}}
	for name, tc := range map[string]struct {
		got  []smapiweb.UpdateResult
		err  error
		want error
	}{
		"all answered":      {known, nil, nil},
		"partly answered":   {known, joined, nil},
		"none answered":     {unknown, joined, first},
		"single failure":    {unknown, second, second},
		"nothing requested": {nil, nil, nil},
	} {
		if got := smapiOutcome(tc.got, tc.err); !errors.Is(got, tc.want) {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}
}
