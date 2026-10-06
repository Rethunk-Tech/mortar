package thunderstore

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func gz(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(raw)
	_ = w.Close()
	return buf.Bytes()
}

type ver struct {
	Description   string   `json:"description"`
	Icon          string   `json:"icon"`
	VersionNumber string   `json:"version_number"`
	Dependencies  []string `json:"dependencies"`
	Downloads     int      `json:"downloads"`
	FileSize      int64    `json:"file_size"`
}

func listing(owner, name, desc string, downloads int, dep, nsfw bool) map[string]any {
	return map[string]any{
		"name": name, "owner": owner, "package_url": "https://x/p/" + owner + "/" + name + "/", "date_updated": "2026-01-01",
		"rating_score": 3, "is_deprecated": dep, "has_nsfw_content": nsfw,
		"versions": []ver{
			{Description: desc, VersionNumber: "2.0.0", Dependencies: []string{"BepInEx-BepInExPack-5.4.2100"}, Downloads: downloads, FileSize: 99},
			{Description: desc, VersionNumber: "1.0.0", Downloads: 1},
		},
	}
}

type fake struct {
	srv                 *httptest.Server
	mux                 *http.ServeMux
	indexHits, chunkHit atomic.Int32
	chunk0, chunk1      []map[string]any
	third, evil         atomic.Bool
}

func newFake(t *testing.T) *fake {
	t.Helper()
	f := &fake{
		chunk0: []map[string]any{
			listing("Alice", "MoreCompany", "more players", 500, false, false),
			listing("Bob", "Cheaty", "more company helper", 9000, false, false),
			listing("Eve", "OldMod", "gone", 99999, true, false),
		},
		chunk1: []map[string]any{
			listing("Carol", "Spicy", "adult stuff", 99999, false, true),
			listing("Dave", "Library", "company library", 100, false, false),
		},
	}
	f.chunk0[0]["categories"] = []string{"Items"}
	f.chunk0[1]["categories"] = []string{"Items", "Cheats"}
	f.chunk1[1]["categories"] = []string{"Libraries"}
	mux := http.NewServeMux()
	f.mux = mux
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	mux.HandleFunc("/c/lethal-company/api/v1/package-listing-index/", func(w http.ResponseWriter, r *http.Request) {
		f.indexHits.Add(1)
		if r.Header.Get("User-Agent") != "Mortar/1.2.3 (+https://mortar.rethunk.tech)" {
			t.Errorf("user agent %q", r.Header.Get("User-Agent"))
		}
		urls := []string{f.srv.URL + "/chunk/0", f.srv.URL + "/chunk/1"}
		if f.evil.Load() {
			urls = []string{"http://169.254.169.254/latest/meta-data"}
		}
		if f.third.Load() {
			urls = append(urls, f.srv.URL+"/chunk/2")
		}
		_, _ = w.Write(gz(t, urls))
	})
	mux.HandleFunc("/chunk/0", func(w http.ResponseWriter, _ *http.Request) { f.chunkHit.Add(1); _, _ = w.Write(gz(t, f.chunk0)) })
	mux.HandleFunc("/chunk/1", func(w http.ResponseWriter, _ *http.Request) { f.chunkHit.Add(1); _, _ = w.Write(gz(t, f.chunk1)) })
	mux.HandleFunc("/chunk/2", func(w http.ResponseWriter, _ *http.Request) {
		f.chunkHit.Add(1)
		_, _ = w.Write(gz(t, []map[string]any{listing("Zed", "NewOne", "fresh", 1, false, false)}))
	})
	return f
}

func names(p source.Page) []string {
	var out []string
	for _, i := range p.Items {
		out = append(out, i.Name)
	}
	return out
}

func TestSearchRanksAndExcludes(t *testing.T) {
	f := newFake(t)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	q := source.Query{Game: "lethal-company", Key: "lethal-company", Text: "company", Page: 1, Version: "1.2.3"}
	p, err := d.Search(t.Context(), q)
	if err != nil {
		t.Fatal(err)
	}
	// MoreCompany matches by name, Cheaty and Library by summary (Cheaty has more downloads); OldMod is deprecated (marked Obsolete, not left out), and Spicy matches nothing here.
	if got := names(p); len(got) != 3 || got[0] != "MoreCompany" || got[1] != "Cheaty" || got[2] != "Library" || p.Total != 3 {
		t.Fatalf("got %v total %d", got, p.Total)
	}
	if p.Items[0].Version != "2.0.0" || p.Items[0].Downloads != 501 || p.Items[0].Author != "Alice" {
		t.Fatalf("item %+v", p.Items[0])
	}
	q.Text, q.Page = "", 1
	if p, _ = d.Search(t.Context(), q); p.Total != 5 || p.Items[0].Name != "OldMod" || !p.Items[0].Obsolete {
		t.Fatalf("browse %v", names(p))
	}
	q.Page = 2
	if p, _ = d.Search(t.Context(), q); len(p.Items) != 0 || p.Total != 5 {
		t.Fatalf("page 2 %v", names(p))
	}
}

func TestCacheHitThenRefresh(t *testing.T) {
	f := newFake(t)
	now := time.Now()
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	q := source.Query{Key: "lethal-company", Text: "library", Page: 1, Version: "1.2.3"}
	if _, err := d.Search(t.Context(), q); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Minute)
	if _, err := d.Search(t.Context(), q); err != nil || f.indexHits.Load() != 1 || f.chunkHit.Load() != 2 {
		t.Fatalf("hit: err %v index %d chunks %d", err, f.indexHits.Load(), f.chunkHit.Load())
	}
	now = now.Add(time.Hour)
	if _, err := d.Search(t.Context(), q); err != nil || f.indexHits.Load() != 2 || f.chunkHit.Load() != 2 {
		t.Fatalf("unchanged index must not refetch chunks: err %v index %d chunks %d", err, f.indexHits.Load(), f.chunkHit.Load())
	}
}

func TestChangedIndexRebuilds(t *testing.T) {
	f := newFake(t)
	now := time.Now()
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	q := source.Query{Key: "lethal-company", Text: "newone", Page: 1, Version: "1.2.3"}
	if p, _ := d.Search(t.Context(), q); p.Total != 0 {
		t.Fatal("unexpected hit")
	}
	f.third.Store(true)
	now = now.Add(2 * time.Hour)
	if p, err := d.Search(t.Context(), q); err != nil || p.Total != 1 {
		t.Fatalf("after refresh: %v %v", err, p.Total)
	}
}

func TestIndexCannotSendMortarElsewhere(t *testing.T) {
	f := newFake(t)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	f.evil.Store(true)
	q := source.Query{Key: "lethal-company", Page: 1, Version: "1.2.3"}
	if _, err := d.Search(t.Context(), q); err == nil {
		t.Fatal("followed a chunk URL on another host")
	}
	q.Key = "../../etc"
	if _, err := d.Search(t.Context(), q); err == nil {
		t.Fatal("accepted a community key that is a path")
	}
}

func TestCategoryFilterSortAndList(t *testing.T) {
	f := newFake(t)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	q := source.Query{Game: "lethal-company", Key: "lethal-company", Page: 1, Version: "1.2.3", Categories: []string{"items"}, Sort: source.SortDownloads}
	p, err := d.Search(t.Context(), q)
	if err != nil || len(p.Items) != 2 || p.Items[0].Name != "Cheaty" {
		t.Fatalf("include items by downloads: %v %v", names(p), err)
	}
	q = source.Query{Game: "lethal-company", Key: "lethal-company", Page: 1, Version: "1.2.3", ExcludeCategories: []string{"Cheats"}, Sort: source.SortName}
	if p, err = d.Search(t.Context(), q); err != nil || len(p.Items) != 4 || p.Items[0].Name != "Library" {
		t.Fatalf("exclude cheats by name: %v %v", names(p), err)
	}
	got, err := d.Categories(t.Context(), "lethal-company")
	if err != nil || len(got) != 3 || got[0] != "Cheats" {
		t.Fatalf("categories %v %v", got, err)
	}
}

func TestDeprecatedListsPackagesAndTheirNamedReplacement(t *testing.T) {
	f := newFake(t)
	f.chunk1 = append(f.chunk1, listing("Fay", "Legacy", "Deprecated: use Alice-MoreCompany instead", 5, true, false),
		listing("Alice", "MoreCompanyOld", "[Deprecated, use MoreCompany instead!]", 5, true, false),
		listing("Gus", "Horn", "Airhorn is replaced with MoreCompany", 5, true, false))
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Deprecated(t.Context(), "lethal-company", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["eve-oldmod"]; !ok || got["eve-oldmod"].Replacement != "" {
		t.Fatalf("a deprecated package without a pointer has no replacement: %+v", got)
	}
	if got["alice-morecompanyold"].Replacement != "Alice-MoreCompany" {
		t.Fatalf("a bare name of the author's own package is the replacement: %+v", got)
	}
	if got["gus-horn"].Replacement != "" {
		t.Fatalf("a bare name of another author's package is not: %+v", got)
	}
	if got["fay-legacy"].Replacement != "Alice-MoreCompany" || len(got) != 4 {
		t.Fatalf("got %+v", got)
	}
}
