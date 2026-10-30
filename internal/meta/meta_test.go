package meta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-smapiweb"
)

// server replays the fixtures and counts requests; down makes every answer a 503.
func server(t *testing.T, down *atomic.Bool, hits *atomic.Int32) *Client {
	t.Helper()
	files := map[string][]byte{
		"/index":              []byte(`{"Pathoschild.ContentPatcher":["CurseForge:309243","Nexus:1915","Nexus:12968","Nexus:50179"],"001":["Nexus:17083","Nexus:17102"]}`),
		"/data/1/1915.json":   []byte(`{"Id":1915,"Name":"Content Patcher","Downloads":[{"Id":146548,"Type":"Main","FileName":"Content Patcher 2.8.1.zip"},{"Id":150000,"Type":"Main","FileName":"Content Patcher 2.9.1.zip"}]}`),
		"/updates":            []byte(`[{"id":"Pathoschild.ContentPatcher","suggestedUpdate":{"version":"2.9.1","url":"https://www.nexusmods.com/stardewvalley/mods/1915"},"metadata":{"gitHubRepo":"Pathoschild/StardewMods","compatibilityStatus":"Ok"}},{"id":"Nope.Nothing"}]`),
		"/data/22/22743.json": []byte(`{"ID":22743,"Downloads":[{"ID":175656,"Type":"Main","FileName":"4a/35/fc/4a35fc45-ad1a-40a7-aa11-3eeab1cbd426"},{"ID":170000,"Type":"Main","FileName":"Alchemistry-22743-2-0-1.zip"}]}`),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		body, ok := files[r.URL.Path]
		switch {
		case down.Load():
			w.WriteHeader(http.StatusServiceUnavailable)
		case !ok:
			w.WriteHeader(http.StatusNotFound)
		default:
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), IndexURL: srv.URL + "/index", PageBase: srv.URL + "/data", UpdatesURL: srv.URL + "/updates"}
}

func TestLookupIgnoresCaseAndCaches(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	refs, err := c.Lookup(context.Background(), "PATHOSCHILD.contentpatcher")
	if err != nil || len(refs) != 4 || refs[1] != (smapiweb.Ref{Site: "Nexus", ID: 1915}) {
		t.Fatalf("refs = %v, err = %v", refs, err)
	}
	if refs, _ := c.Lookup(context.Background(), "nobody.nothing"); refs != nil {
		t.Fatalf("unknown id gave %v", refs)
	}
	// A second client shares only the disk cache.
	c2 := &Client{HTTP: c.HTTP, CacheDir: c.CacheDir, IndexURL: c.IndexURL}
	down.Store(true)
	before := hits.Load()
	if refs, err := c2.Lookup(context.Background(), "001"); err != nil || len(refs) != 2 || hits.Load() != before {
		t.Fatalf("cache hit failed: %v %v hits %d->%d", refs, err, before, hits.Load())
	}
}

func TestIndexStaleFallbackAndFailure(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	now := time.Now()
	c.Now = func() time.Time { return now }
	if _, err := c.Lookup(context.Background(), "001"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(60 * 24 * time.Hour)
	c2 := &Client{HTTP: c.HTTP, CacheDir: c.CacheDir, IndexURL: c.IndexURL, Now: c.Now}
	down.Store(true)
	if refs, err := c2.Lookup(context.Background(), "001"); err != nil || len(refs) != 2 {
		t.Fatalf("stale fallback: %v %v", refs, err)
	}
	fresh := &Client{HTTP: c.HTTP, CacheDir: t.TempDir(), IndexURL: c.IndexURL}
	if _, err := fresh.Lookup(context.Background(), "001"); err == nil {
		t.Fatal("no cache and no network must error")
	}
}

func TestPageIsCachedAndAMissingOneErrors(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	p, err := c.Page(context.Background(), 1915)
	if err != nil || p.Name != "Content Patcher" || len(p.Downloads) != 2 {
		t.Fatalf("page = %+v, err = %v", p, err)
	}
	down.Store(true)
	before := hits.Load()
	if _, err := c.Page(context.Background(), 1915); err != nil || hits.Load() != before {
		t.Fatalf("cache hit failed: %v", err)
	}
	if _, err := c.Page(context.Background(), 3999); err == nil {
		t.Fatal("missing page must error")
	}
}

func TestPageDropsStoragePathFileNames(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	for range 2 { // fetched, then from the cache
		p, err := c.Page(context.Background(), 22743)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Downloads) != 2 || p.Downloads[0].FileName != "" || p.Downloads[1].FileName != "Alchemistry-22743-2-0-1.zip" {
			t.Fatalf("downloads = %+v", p.Downloads)
		}
	}
}

var cpRequest = UpdateRequest{APIVersion: "4.5.2", GameVersion: "1.6.15", Platform: "Windows", Mods: []smapiweb.InstalledMod{
	{ID: "Pathoschild.ContentPatcher", UpdateKeys: []string{"Nexus:1915"}, Version: "2.0.0"},
	{ID: "Nope.Nothing", Version: "1.0"},
}}

func TestCheckUpdatesRealFixture(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	got := c.CheckUpdates(context.Background(), cpRequest)
	cp := got[0]
	if !cp.Known || cp.Suggested == nil || cp.Suggested.Version != "2.9.1" || cp.Compatibility != "Ok" || cp.GitHubRepo != "Pathoschild/StardewMods" {
		t.Fatalf("content patcher = %+v", cp)
	}
	if !got[1].Known || got[1].Suggested != nil {
		t.Fatalf("unlisted mod = %+v", got[1])
	}
	down.Store(true)
	before := hits.Load()
	again := c.CheckUpdates(context.Background(), cpRequest)
	if again[0].Suggested == nil || hits.Load() != before {
		t.Fatalf("cache hit failed: %+v", again[0])
	}
}

func TestCheckUpdatesStaleThenUnknown(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	now := time.Now()
	c.Now = func() time.Time { return now }
	c.CheckUpdates(context.Background(), cpRequest)
	now = now.Add(2 * time.Hour)
	down.Store(true)
	got := c.CheckUpdates(context.Background(), cpRequest)
	if !got[0].Known || got[0].Suggested == nil {
		t.Fatalf("stale fallback lost the answer: %+v", got[0])
	}
	if !got[1].Known {
		t.Fatalf("stale fallback lost the no-update answer: %+v", got[1])
	}
	cold := &Client{HTTP: c.HTTP, CacheDir: t.TempDir(), UpdatesURL: c.UpdatesURL}
	for _, r := range cold.CheckUpdates(context.Background(), cpRequest) {
		if r.Known || r.Suggested != nil || r.ID == "" {
			t.Fatalf("failure must give unknown, got %+v", r)
		}
	}
}

func TestCheckUpdatesSharesConcurrentAndPausesAfterFailure(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() { c.CheckUpdates(context.Background(), cpRequest) })
	}
	wg.Wait()
	if n := hits.Load(); n != 1 {
		t.Fatalf("concurrent checks of the same mods asked %d times, want 1", n)
	}

	cold := &Client{HTTP: c.HTTP, CacheDir: t.TempDir(), UpdatesURL: c.UpdatesURL}
	now := time.Now()
	cold.Now = func() time.Time { return now }
	down.Store(true)
	cold.CheckUpdates(context.Background(), cpRequest)
	before := hits.Load()
	now = now.Add(time.Minute)
	cold.CheckUpdates(context.Background(), cpRequest)
	if hits.Load() != before {
		t.Fatal("a failed ask must pause further asks")
	}
	down.Store(false)
	now = now.Add(updatesBackoff)
	if got := cold.CheckUpdates(context.Background(), cpRequest); !got[0].Known {
		t.Fatalf("ask after the pause should succeed: %+v", got[0])
	}
}

func TestPageMissingFromDatasetIsCached(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.NotFound(w, nil)
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), PageBase: srv.URL}
	for range 2 {
		if _, err := c.Page(context.Background(), 4216); err != nil {
			t.Fatalf("a page the dataset lacks is an answer, not an error: %v", err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("missing page fetched %d times, want once", hits.Load())
	}
}

func TestNewerOrdersPrereleasesAndPrefixes(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"1.2.0", "1.2.0-beta.1", true},
		{"1.2.0-beta.2", "1.2.0", false},
		{"4.1.0", "4.1.0-beta.1", true},
		{"v4.10.0", "4.9.9", true},
		{"4.5.2", "4.5.2", false},
		{"4.5", "4.5.1", false},
		{"not a version", "1.0.0", false},
	} {
		if got := Newer(tc.a, tc.b); got != tc.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
