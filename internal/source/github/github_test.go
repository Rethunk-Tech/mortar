package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	_, err := (&Driver{URL: srv.URL, Web: srv.URL}).Search(context.Background(), source.Query{Key: "k", Text: "x", Page: 1})
	if !errors.Is(err, source.ErrBusy) || usererr.KindOf(err) != usererr.RateLimited {
		t.Fatalf("got %v", err)
	}
}

func TestSearchWaitsOutARefusal(t *testing.T) {
	t.Parallel()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
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
	d := &Driver{URL: srv.URL, Web: srv.URL, Now: func() time.Time { return now }, Topics: func(string) []string { return []string{"a", "b", "c"} }}
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
	d := &Driver{URL: srv.URL, Web: srv.URL, Topics: func(string) []string { return []string{"a", "b"} }}
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
	d = &Driver{URL: srv.URL, Web: srv.URL, Topics: d.Topics}
	if page, err := d.Search(context.Background(), source.Query{Key: "a", Page: 1}); err != nil || len(page.Items) != 2 || fmt.Sprint(page.Failed) != "[topic:b]" {
		t.Fatalf("one topic refused: %d items, failed %v, err %v", len(page.Items), page.Failed, err)
	}
	mu.Lock()
	refuse["topic:a"] = true
	mu.Unlock()
	d = &Driver{URL: srv.URL, Web: srv.URL, Topics: d.Topics}
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
	d := &Driver{URL: srv.URL, Web: srv.URL, Topics: func(string) []string { return []string{"a", "b"} }}
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

func TestSearchFallsBackToTheSearchPageAsMortar(t *testing.T) {
	t.Parallel()
	var pages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if !strings.HasPrefix(r.UserAgent(), "Mortar") || r.URL.Query().Get("q") != "x y topic:a" {
			t.Errorf("asked %q as %q", r.URL.RawQuery, r.UserAgent())
		}
		pages = append(pages, r.URL.Query().Get("p"))
		page, n := 1, 10
		if r.URL.Query().Get("p") == "2" {
			page, n = 2, 2
		}
		var rows []string
		for i := range n {
			rows = append(rows, fmt.Sprintf(`{"hl_trunc_description":"A <em>x</em> &amp; more","followers":%d,"repo":{"repository":{"name":"r%d-%d","owner_id":7,"owner_login":"o"}}}`, 50-i, page, i))
		}
		fmt.Fprintf(w, `{"payload":{"blackbirdSearchRoute":{"result_count":12,"results":[%s]}}}`, strings.Join(rows, ","))
	}))
	t.Cleanup(srv.Close)
	d := &Driver{URL: srv.URL, Web: srv.URL, webGap: time.Nanosecond}
	page, err := d.Search(context.Background(), source.Query{Key: "a", Text: "x y", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 12 || page.Total != 12 || strings.Join(pages, ",") != "1,2" {
		t.Fatalf("%d items of %d from pages %v", len(page.Items), page.Total, pages)
	}
	if it := page.Items[0]; it.ID != "o/r1-0" || it.Summary != "A x & more" || it.Stars != 50 || it.Picture != "https://avatars.githubusercontent.com/u/7?v=4" {
		t.Fatalf("first %+v", it)
	}
}

func TestSearchAsksGraphQLOnceForEveryTopicAndCachesItsPages(t *testing.T) {
	t.Parallel()
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			asked = append(asked, r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		var body struct {
			Query     string            `json:"query"`
			Variables map[string]string `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		asked = append(asked, "graphql")
		if strings.Contains(body.Query, "repositoryTopics") || body.Variables["q1"] != "topic:b sort:stars" {
			t.Errorf("query %s %v", body.Query, body.Variables)
		}
		node := func(name string, stars int) string {
			return fmt.Sprintf(`{"nameWithOwner":"o/%s","name":"%s","stargazerCount":%d,"owner":{"login":"o"}}`, name, name, stars)
		}
		var a []string
		for i := range 25 {
			a = append(a, node(fmt.Sprintf("a%d", i), 100-i))
		}
		fmt.Fprintf(w, `{"data":{"s0":{"repositoryCount":25,"nodes":[%s]},"s1":{"repositoryCount":1,"nodes":[%s]}}}`,
			strings.Join(a, ","), node("b", 1000))
	}))
	t.Cleanup(srv.Close)
	d := &Driver{URL: srv.URL, LoggedIn: func(context.Context) bool { return true }, Topics: func(string) []string { return []string{"a", "b"} }}
	page, err := d.Search(context.Background(), source.Query{Key: "a", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 20 || page.Items[0].ID != "o/b" {
		t.Fatalf("page 1: %d items, first %+v", len(page.Items), page.Items)
	}
	if page, err = d.Search(context.Background(), source.Query{Key: "a", Page: 2}); err != nil || len(page.Items) != 6 {
		t.Fatalf("page 2: %d items, %v", len(page.Items), err)
	}
	if strings.Join(asked, ",") != "graphql" {
		t.Fatalf("asked %v", asked)
	}
}
