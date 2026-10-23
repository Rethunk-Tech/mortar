package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestSearchQueryAndPaging(t *testing.T) {
	t.Parallel()
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"total_count":3,"items":[{"full_name":"Pathoschild/SMAPI","name":"SMAPI","description":"d","html_url":"https://github.com/Pathoschild/SMAPI","stargazers_count":9,"updated_at":"u","owner":{"login":"Pathoschild","avatar_url":"a"}}]}`))
	}))
	t.Cleanup(srv.Close)
	d := &Driver{URL: srv.URL}
	page, err := d.Search(context.Background(), source.Query{Key: "stardew-valley-mod", Text: "smapi tools", Page: 2})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Items) != 1 || page.Items[0].Stars != 9 || page.Items[0].ID != "Pathoschild/SMAPI" ||
		page.Items[0].Name != "SMAPI" {
		t.Fatalf("page %+v", page)
	}
	if !strings.Contains(gotURL, "topic%3Astardew-valley-mod") {
		t.Fatalf("topic missing: %s", gotURL)
	}
	if strings.Contains(gotURL, "sort=") || !strings.Contains(gotURL, "per_page=20") || !strings.Contains(gotURL, "page=2") {
		t.Fatalf("paging %s", gotURL)
	}
}

func TestSortParamNeverMapsDownloadsToStars(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ sort, text, want string }{
		{source.SortStars, "x", "stars"},
		{source.SortForks, "x", "forks"},
		{source.SortUpdated, "", "updated"},
		{"", "x", ""},
		{"", " ", "stars"},
		{source.SortDownloads, "x", ""},
	} {
		if got := sortParam(c.sort, c.text); got != c.want {
			t.Errorf("sortParam(%q, %q) = %q, want %q", c.sort, c.text, got, c.want)
		}
	}
}

func TestSearchRateLimit(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	_, err := (&Driver{URL: srv.URL}).Search(context.Background(), source.Query{Key: "k", Text: "x", Page: 1})
	if !errors.Is(err, source.ErrBusy) {
		t.Fatalf("got %v", err)
	}
}

func TestSearchWaitsOutARefusal(t *testing.T) {
	t.Parallel()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			w.Header().Set("X-Ratelimit-Reset", "1700000030")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	}))
	t.Cleanup(srv.Close)
	now := time.Unix(1_700_000_000, 0).UTC()
	d := &Driver{URL: srv.URL, Now: func() time.Time { return now }, Topics: func(string) []string { return []string{"a", "b", "c"} }}
	q := source.Query{Key: "a", Text: "x y", Page: 1}
	for range 3 {
		if _, err := d.Search(context.Background(), q); !errors.Is(err, source.ErrBusy) {
			t.Fatalf("got %v", err)
		}
	}
	if hits != 1 {
		t.Fatalf("asked GitHub %d times while refused", hits)
	}
	now = now.Add(30 * time.Second)
	if _, err := d.Search(context.Background(), q); err != nil {
		t.Fatal(err)
	}
}

func TestSearchCaches(t *testing.T) {
	t.Parallel()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	}))
	t.Cleanup(srv.Close)
	now := time.Unix(1_700_000_000, 0).UTC()
	d := &Driver{URL: srv.URL, Now: func() time.Time { return now }}
	for _, q := range []source.Query{{Key: "k", Text: "x y", Page: 1}, {Key: "k", Text: "x y", Page: 1}, {Text: "x y", Page: 1}} {
		if _, err := d.Search(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	// The second query is cached and the third has no topic, so neither reaches GitHub.
	if hits != 1 {
		t.Fatalf("hits %d", hits)
	}
}

func TestSearchMergesTopicsOnceEachByStars(t *testing.T) {
	t.Parallel()
	refuse := map[string]bool{}
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		mu.Lock()
		refused := refuse[q]
		mu.Unlock()
		switch {
		case refused:
			w.WriteHeader(http.StatusForbidden)
		case q == "topic:a":
			_, _ = w.Write([]byte(`{"total_count":2,"items":[{"full_name":"o/shared","stargazers_count":5},{"full_name":"o/low","stargazers_count":1}]}`))
		case q == "topic:b":
			_, _ = w.Write([]byte(`{"total_count":2,"items":[{"full_name":"o/top","stargazers_count":9},{"full_name":"O/Shared","stargazers_count":5}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	d := &Driver{URL: srv.URL, Topics: func(string) []string { return []string{"a", "b"} }}
	page, err := d.Search(context.Background(), source.Query{Key: "a", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, it := range page.Items {
		ids = append(ids, it.ID)
	}
	if strings.Join(ids, " ") != "o/top o/shared o/low" || page.Total != 3 {
		t.Fatalf("merged %v, total %d", ids, page.Total)
	}

	mu.Lock()
	refuse["topic:b"] = true
	mu.Unlock()
	d = &Driver{URL: srv.URL, Topics: d.Topics}
	if page, err := d.Search(context.Background(), source.Query{Key: "a", Page: 1}); err != nil || len(page.Items) != 2 {
		t.Fatalf("one topic refused: %d items, err %v", len(page.Items), err)
	}
	mu.Lock()
	refuse["topic:a"] = true
	mu.Unlock()
	d = &Driver{URL: srv.URL, Topics: d.Topics}
	if _, err := d.Search(context.Background(), source.Query{Key: "a", Page: 1}); !errors.Is(err, source.ErrBusy) {
		t.Fatalf("every topic refused: %v", err)
	}
}

func TestSearchTakesAWordAsAnOwnerToo(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("q") {
		case "user:Org":
			_, _ = w.Write([]byte(`{"total_count":2,"items":[{"full_name":"Org/mod","topics":["b"]},{"full_name":"Org/site","topics":["web"]}]}`))
		case "user:nobody":
			w.WriteHeader(http.StatusUnprocessableEntity)
		default:
			_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	d := &Driver{URL: srv.URL, Topics: func(string) []string { return []string{"a", "b"} }}
	page, err := d.Search(context.Background(), source.Query{Key: "a", Text: "Org", Page: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "Org/mod" {
		t.Fatalf("owner search: %+v, %v", page.Items, err)
	}
	if page, err := d.Search(context.Background(), source.Query{Key: "a", Text: "nobody", Page: 1}); err != nil || len(page.Items) != 0 {
		t.Fatalf("no such owner: %+v, %v", page.Items, err)
	}
}

// TestLiveLethalCompanyTopics is the check that the catalog's topics reach the search, since LethalCompanyModding tags
// its mods lethalcompany-mod and lethal-company-community, never the key's lethal-company-mod, and that its name finds
// them.
func TestLiveLethalCompanyTopics(t *testing.T) {
	if os.Getenv("MORTAR_LIVE_SOURCES") == "" {
		t.Skip("set MORTAR_LIVE_SOURCES=1 to query GitHub")
	}
	g, ok := components.Game("lethal-company")
	if !ok {
		t.Fatal("catalog has no lethal-company")
	}
	gs, _ := g.Source("github")
	d := &Driver{}
	for _, c := range []struct{ text, want string }{
		{"Enhancer", "Enhancer"},
		{"UnityVertexLibrary", "UnityVertexLibrary"},
		{"LethalCompanyModding", "Enhancer"},
		{"LethalCompanyModding", "UnityVertexLibrary"},
	} {
		page, err := d.Search(t.Context(), source.Query{Game: "lethal-company", Key: gs.Key, Text: c.text, Page: 1})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(page.Items, func(it source.Item) bool { return it.ID == "LethalCompanyModding/"+c.want }) {
			t.Errorf("%s: %s missing from %d repositories", c.text, c.want, len(page.Items))
		}
	}
}
