package vsmoddb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func fake(t *testing.T) Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mods":
			if r.URL.Query().Get("text") != "" && r.URL.Query().Get("orderby") != "downloads" {
				http.Error(w, "a search with text and no sort must order by downloads", http.StatusBadRequest)
				return
			}
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
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
	first, err := d.Search(context.Background(), source.Query{Page: source.FirstPage, ExcludeCategories: []string{"library"}})
	if err != nil {
		t.Fatal(err)
	}
	// A full page means more may follow, so the total offers one more.
	if first.Total != source.PageSize+1 || first.Hidden != 1 || first.Items[0].URL != siteURL+"/bee" || first.Items[0].Endorsements != 7 {
		t.Fatalf("first page = %+v", first)
	}
	second, err := d.Search(context.Background(), source.Query{Page: source.FirstPage + 1, Text: "b"})
	if err != nil || len(second.Items) != source.PageSize || second.Items[0].ID != "2-22" {
		t.Fatalf("second page = %+v, %v", second, err)
	}
}

func TestResolvePrefersTheNewestStableReleaseAndMatchesAReleaseID(t *testing.T) {
	d := fake(t)
	for _, tc := range []struct{ version, want string }{{"", "1.14.3"}, {"3", "2.0.0-rc.1"}, {"1.14.3", "1.14.3"}} {
		r, err := d.Resolve(context.Background(), "890-4405", tc.version)
		if err != nil || r.Version != tc.want || r.ID != "890-4405" {
			t.Fatalf("Resolve(%q) = %+v, %v", tc.version, r, err)
		}
	}
	if _, err := d.Resolve(context.Background(), "890", "9.9"); err == nil {
		t.Fatal("a release the mod lacks resolved")
	}
	items, err := d.Items(context.Background(), "", "", []string{"890-4405", "nope"})
	if err != nil || items["890-4405"].Version != "1.14.3" || len(items) != 1 {
		t.Fatalf("items = %+v, %v", items, err)
	}
	if got := (Driver{}).ModPageURL("", "890-4405"); got != siteURL+"/show/mod/4405" || (Driver{}).ModPageURL("", "890") != "" {
		t.Fatalf("page = %q", got)
	}
}

func TestCategoriesHandsOutACopyOfTheCachedTags(t *testing.T) {
	tags.Lock()
	tags.ids, tags.names = map[string]string{"library": "1"}, []string{"Library"}
	tags.Unlock()
	got, err := Driver{}.Categories(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	got[0] = "changed"
	if again, _ := (Driver{}).Categories(context.Background(), ""); again[0] != "Library" {
		t.Fatalf("the cache changed: %v", again)
	}
}

func TestSearchOfAnUnknownCategoryIsEmpty(t *testing.T) {
	tags.Lock()
	tags.ids, tags.names = map[string]string{"library": "1"}, []string{"Library"}
	tags.Unlock()
	got, err := fake(t).Search(context.Background(), source.Query{Page: source.FirstPage, Categories: []string{"nope"}})
	if err != nil || len(got.Items) != 0 || got.Total != 0 {
		t.Fatalf("page = %+v, %v", got, err)
	}
}
