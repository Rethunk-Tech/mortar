package meta

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := fsx.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// server replays the fixtures and counts requests; down makes every answer a 503.
func server(t *testing.T, down *atomic.Bool, hits *atomic.Int32) *Client {
	t.Helper()
	files := map[string][]byte{
		"/index":              fixture(t, "index.json"),
		"/data/1/1915.json":   fixture(t, "nexus-1915.json"),
		"/updates":            fixture(t, "smapi-mods.json"),
		"/data/22/22743.json": []byte(`{"ID":22743,"Downloads":[{"ID":175656,"Type":"Main","FileName":"4a/35/fc/4a35fc45-ad1a-40a7-aa11-3eeab1cbd426"},{"ID":170000,"Type":"Main","FileName":"Alchemistry-22743-2-0-1.zip"}]}`),
		"/data/2/2000.json":   []byte(`{"Id":"2000","Name":7,"Downloads":[null,{"Id":"x","Version":1.5,"SizeInBytes":"big","Mods":[{"Manifest":{"UniqueID":"A.B","Version":2,"UpdateKeys":"Nexus:2000","Dependencies":[{"UniqueID":"C.D","IsRequired":false},{"UniqueID":"E.F","MinimumVersion":"1.0"}],"ContentPackFor":{"UniqueID":"G.H"}}},{"Manifest":{"Name":"no id"}},{}]}]}`),
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
	if err != nil || len(refs) != 4 || refs[1] != (Ref{Site: "Nexus", ID: 1915}) {
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

func TestParseIndexSkipsOddEntries(t *testing.T) {
	got, err := parseIndex([]byte(`{"A.B":["Nexus:5","Nexus:x","junk",3,"Nexus:6"],"C":"Nexus:1","D":null}`))
	if err != nil || len(got) != 1 || len(got["a.b"]) != 2 || got["a.b"][1].ID != 6 {
		t.Fatalf("got %v, err %v", got, err)
	}
	if _, err := parseIndex([]byte(`[]`)); err == nil {
		t.Fatal("array is not an index")
	}
}

func TestPageRealFixture(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	p, err := c.Page(context.Background(), 1915)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Content Patcher" || len(p.Downloads) != 2 {
		t.Fatalf("page = %+v", p)
	}
	f := p.Downloads[1]
	if f.SizeInBytes != 389967 || f.Type != "Main" || f.Mods[0].UniqueID != "Pathoschild.ContentPatcher" || f.Mods[0].Version != "2.9.1" || f.Mods[0].UpdateKeys[0] != "Nexus:1915" {
		t.Fatalf("file = %+v", f)
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

func TestPageLenientParse(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	p, err := c.Page(context.Background(), 2000)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Downloads) != 2 || len(p.Downloads[1].Mods) != 1 {
		t.Fatalf("downloads = %+v", p.Downloads)
	}
	m := p.Downloads[1].Mods[0]
	want := []Dependency{{"C.D", "", false}, {"E.F", "1.0", true}, {"G.H", "", true}}
	if m.Version != "2" || len(m.UpdateKeys) != 0 || len(m.Dependencies) != 3 || m.Dependencies[0] != want[0] || m.Dependencies[1] != want[1] || m.Dependencies[2] != want[2] {
		t.Fatalf("mod = %+v", m)
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

var cpRequest = UpdateRequest{APIVersion: "4.5.2", GameVersion: "1.6.15", Platform: "Windows", Mods: []InstalledMod{
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

func TestCheckUpdatesBatchesAndSendsRequest(t *testing.T) {
	var batches atomic.Int32
	var first apiRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req apiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if batches.Add(1) == 1 {
			first = req
		}
		out := make([]apiMod, len(req.Mods))
		for i, m := range req.Mods {
			out[i] = apiMod{ID: m.ID}
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), UpdatesURL: srv.URL}
	req := UpdateRequest{APIVersion: "4.5.2", GameVersion: "1.6.15", Platform: "Linux"}
	for i := range 230 {
		req.Mods = append(req.Mods, InstalledMod{ID: "M" + string(rune('A'+i%26)) + string(rune('a'+i/26)), Version: "1.0"})
	}
	got := c.CheckUpdates(context.Background(), req)
	if batches.Load() != 3 || len(got) != 230 || !got[229].Known || got[229].ID != req.Mods[229].ID {
		t.Fatalf("batches %d, results %d", batches.Load(), len(got))
	}
	if len(first.Mods) != 100 || !first.IncludeExtendedMetadata || first.Platform != "Linux" || first.GameVersion != "1.6.15" || first.APIVersion != "4.5.2" {
		t.Fatalf("request = %+v", first)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0.0", 0},
		{"2.9.1", "2.10.0", -1},
		{"1.0.0", "1.0.0-beta", 1},
		{"1.0.0-beta", "1.0.0-beta.2", -1},
		{"1.0.0-beta.2", "1.0.0-beta.10", -1},
		{"1.0.0-alpha", "1.0.0-1", 1},
		{"1.0.0-RC1", "1.0.0-rc1", 0},
		{"v1.2.3+build5", "1.2.3", 0},
		{"1.2.3.4", "1.2.3", 1},
		{"3.0.0-unofficial.1-pathoschild", "3.0.0", -1},
	}
	for _, c := range cases {
		got, ok := CompareVersions(c.a, c.b)
		if !ok || got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, %v; want %d", c.a, c.b, got, ok, c.want)
		}
		if rev, _ := CompareVersions(c.b, c.a); rev != -c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, not the mirror", c.b, c.a, rev)
		}
	}
	for _, bad := range []string{"", "1", "abc", "1.x"} {
		if _, ok := CompareVersions(bad, "1.0"); ok {
			t.Errorf("%q should not parse", bad)
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

func TestCheckUpdatesUnlistedModIsKnownAndCached(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	req := UpdateRequest{Mods: []InstalledMod{{ID: "Nobody.Lists.This", Version: "1.0.0"}}}
	first := c.CheckUpdates(context.Background(), req)
	if !first[0].Known || first[0].Suggested != nil {
		t.Fatalf("a mod the API answered without listing is known with no update: %+v", first[0])
	}
	asked := hits.Load()
	c.CheckUpdates(context.Background(), req)
	if hits.Load() != asked {
		t.Fatalf("unlisted mod asked again: %d -> %d requests", asked, hits.Load())
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
