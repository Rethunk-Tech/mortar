package vsmoddb

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-vintagestory/moddb"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

var hits atomic.Int32

func fake(t *testing.T) Driver {
	t.Helper()
	hits.Store(0)
	lists.m = nil
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mods":
			if r.URL.Query().Get("text") != "" && r.URL.Query().Get("orderby") != "downloads" {
				http.Error(w, "a search with text and no sort must order by downloads", http.StatusBadRequest)
				return
			}
			hits.Add(1)
			limit := 3*source.PageSize + 1
			mods := `{"modid":1,"assetid":11,"name":"A","tags":["Library"],"lastreleased":"2026-10-16 08:25:06"}`
			for i := 1; i < limit; i++ {
				mods += `,{"modid":2,"assetid":22,"name":"B","urlalias":"bee","follows":7}`
			}
			_, _ = w.Write([]byte(`{"statuscode":"200","mods":[` + mods + `]}`))
		case "/mod/890":
			// The site lists releases newest first, test builds included.
			_, _ = w.Write([]byte(`{"statuscode":"200","mod":{"modid":890,"assetid":4405,"name":"Carry On","releases":[
				{"releaseid":3,"modversion":"2.0.0-rc.1","mainfile":"https://cdn/3.zip","filename":"3.zip"},
				{"releaseid":2,"modversion":"1.14.3","mainfile":"https://cdn/2.zip","filename":"2.zip"}]}}`))
		default:
			_, _ = w.Write([]byte(`{"statuscode":"404"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return Driver{HTTP: srv.Client(), URL: srv.URL}
}

func TestSearchKeepsTheLastPageOfTheLimitAndDropsExcludedTags(t *testing.T) {
	d := fake(t)
	first, err := d.Search(t.Context(), source.Query{Page: source.FirstPage, ExcludeCategories: []string{"library"}})
	if err != nil {
		t.Fatal(err)
	}
	// A full page means more may follow, so the total offers one more.
	if first.Total != source.PageSize+1 || first.Hidden != 1 || first.Items[0].URL != moddb.SiteURL+"/bee" || first.Items[0].Endorsements != 7 {
		t.Fatalf("first page = %+v", first)
	}
	second, err := d.Search(t.Context(), source.Query{Page: source.FirstPage + 1, Text: "b"})
	if err != nil || len(second.Items) != source.PageSize || second.Items[0].ID != "2-22" {
		t.Fatalf("second page = %+v, %v", second, err)
	}
}

func TestResolvePrefersTheNewestStableReleaseAndMatchesAReleaseID(t *testing.T) {
	d := fake(t)
	for _, tc := range []struct{ version, want string }{{"", "1.14.3"}, {"3", "2.0.0-rc.1"}, {"1.14.3", "1.14.3"}} {
		r, err := d.Resolve(t.Context(), "890-4405", tc.version)
		if err != nil || r.Version != tc.want || r.ID != "890-4405" {
			t.Fatalf("Resolve(%q) = %+v, %v", tc.version, r, err)
		}
	}
	if _, err := d.Resolve(t.Context(), "890", "9.9"); err == nil {
		t.Fatal("a release the mod lacks resolved")
	}
	items, err := d.Items(t.Context(), "", "", []string{"890-4405", "nope"})
	if err != nil || items["890-4405"].Version != "1.14.3" || len(items) != 1 {
		t.Fatalf("items = %+v, %v", items, err)
	}
	if got := (Driver{}).ModPageURL("", "890-4405"); got != moddb.SiteURL+"/show/mod/4405" || (Driver{}).ModPageURL("", "890") != "" {
		t.Fatalf("page = %q", got)
	}
}

func TestCategoriesHandsOutACopyOfTheCachedTags(t *testing.T) {
	tags.Lock()
	tags.ids, tags.names, tags.until = map[string]string{"library": "1"}, []string{"Library"}, now().Add(time.Hour)
	tags.Unlock()
	got, err := Driver{}.Categories(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	got[0] = "changed"
	if again, _ := (Driver{}).Categories(t.Context(), ""); again[0] != "Library" {
		t.Fatalf("the cache changed: %v", again)
	}
}

func TestSearchOfAnUnknownCategoryIsEmpty(t *testing.T) {
	tags.Lock()
	tags.ids, tags.names, tags.until = map[string]string{"library": "1"}, []string{"Library"}, now().Add(time.Hour)
	tags.Unlock()
	got, err := fake(t).Search(t.Context(), source.Query{Page: source.FirstPage, Categories: []string{"nope"}})
	if err != nil || len(got.Items) != 0 || got.Total != 0 {
		t.Fatalf("page = %+v, %v", got, err)
	}
}

func TestPagesShareOneFetchUntilTheTTLPasses(t *testing.T) {
	d := fake(t)
	clock := time.Now()
	now = func() time.Time { return clock }
	t.Cleanup(func() { now = time.Now })
	for p := source.FirstPage; p < source.FirstPage+3; p++ {
		if _, err := d.Search(t.Context(), source.Query{Page: p}); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("3 pages made %d fetches, want 1", hits.Load())
	}
	clock = clock.Add(modListTTL + time.Second)
	if _, err := d.Search(t.Context(), source.Query{Page: source.FirstPage}); err != nil || hits.Load() != 2 {
		t.Fatalf("after the TTL: %d fetches, %v; want 2", hits.Load(), err)
	}
}

// A site refusing calls for a while is the queue's busy error, so the item waits instead of failing.
func TestARefusedCallIsTheQueuesBusyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	d := Driver{HTTP: srv.Client(), URL: srv.URL}
	lists.m = nil
	if _, err := d.Search(t.Context(), source.Query{Page: source.FirstPage}); !errors.Is(err, source.ErrBusy) {
		t.Fatalf("search: %v, want ErrBusy", err)
	}
	if _, err := d.Resolve(t.Context(), "890", ""); !errors.Is(err, source.ErrBusy) {
		t.Fatalf("resolve: %v, want ErrBusy", err)
	}
	tags.Lock()
	tags.ids, tags.until = nil, time.Time{}
	tags.Unlock()
	if _, err := d.Categories(t.Context(), ""); !errors.Is(err, source.ErrBusy) {
		t.Fatalf("categories: %v, want ErrBusy", err)
	}
}
