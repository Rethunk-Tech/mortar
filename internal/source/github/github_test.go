package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	page, err := d.Search(context.Background(), source.Query{Key: "stardew-valley-mod", Text: "smapi", Page: 2})
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
	if !strings.Contains(gotURL, "sort=stars") || !strings.Contains(gotURL, "per_page=20") || !strings.Contains(gotURL, "page=2") {
		t.Fatalf("paging %s", gotURL)
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
	for _, q := range []source.Query{{Key: "k", Text: "x", Page: 1}, {Key: "k", Text: "x", Page: 1}, {Text: "x", Page: 1}} {
		if _, err := d.Search(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	// The second query is cached and the third has no topic, so neither reaches GitHub.
	if hits != 1 {
		t.Fatalf("hits %d", hits)
	}
}
