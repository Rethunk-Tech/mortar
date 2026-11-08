package nexus

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func searchQuery(t *testing.T, raw string) string {
	t.Helper()
	var body struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	return body.Query
}

func TestSearchQueryAndPaging(t *testing.T) {
	t.Parallel()
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/graphql" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Application-Name") != "Mortar" || r.Header.Get("Application-Version") != "test" {
			t.Errorf("app headers %q %q", r.Header.Get("Application-Name"), r.Header.Get("Application-Version"))
		}
		if r.Header.Get("User-Agent") != "Mortar/test" {
			t.Errorf("User-Agent %q", r.Header.Get("User-Agent"))
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":27,"nodes":[{"modId":1348,"name":"SpaceCore","summary":"s","author":"a","version":"1","endorsements":233359,"downloads":1,"pictureUrl":"p","updatedAt":"2026-01-01T00:00:00Z","adultContent":true,"modCategory":{"name":"Cheats"}}]}}}`))
	}))
	t.Cleanup(srv.Close)
	page, err := Driver{URL: srv.URL}.Search(t.Context(), source.Query{Game: "stardew", Key: "stardewvalley", Text: "space", Page: 2, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 27 || len(page.Items) != 1 || page.Items[0].ID != "1348" || page.Items[0].Source != "nexus" || !page.Items[0].Adult || page.Items[0].Category != "Cheats" || page.Items[0].Updated != "2026-01-01T00:00:00Z" {
		t.Fatalf("page %+v", page)
	}
	if page.Items[0].URL != "https://www.nexusmods.com/stardewvalley/mods/1348" {
		t.Fatalf("url %s", page.Items[0].URL)
	}
	query := searchQuery(t, gotBody)
	for _, want := range []string{
		`gameDomainName:[{value:"stardewvalley"}]`,
		`name:[{value:"space", op:WILDCARD}]`,
		"sort:[{endorsements:{direction:DESC}}]",
		"count: 20, offset: 20",
		"adultContent",
		"modCategory { name }",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("%q missing: %s", want, query)
		}
	}
}

func TestSearchNeedsTheGameKey(t *testing.T) {
	t.Parallel()
	_, err := (Driver{}).Search(t.Context(), source.Query{Game: "g", Page: 1})
	if err == nil || err.Error() != `game "g" has no Nexus domain` {
		t.Fatalf("got %v", err)
	}
}

func TestSearchTextCannotEscapeItsStringLiteral(t *testing.T) {
	t.Parallel()
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":0,"nodes":[]}}}`))
	}))
	t.Cleanup(srv.Close)
	_, err := Driver{URL: srv.URL}.Search(t.Context(), source.Query{Key: "stardewvalley", Text: "a\" }) { x } #\\\x01", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	query := searchQuery(t, gotBody)
	if want := `op:WILDCARD`; !strings.Contains(query, `{value:"a\" }) { x } #\\\u0001", `+want) {
		t.Fatalf("text not carried as one escaped literal: %s", query)
	}
}

func TestEmptySearchListsMostEndorsedWithoutNameFilter(t *testing.T) {
	t.Parallel()
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":0,"nodes":[]}}}`))
	}))
	t.Cleanup(srv.Close)
	if _, err := (Driver{URL: srv.URL}).Search(t.Context(), source.Query{Game: "g", Key: "gk", Page: 1}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(gotBody, "name:") || !strings.Contains(gotBody, "endorsements") {
		t.Fatalf("query %s", gotBody)
	}
}

func TestCategoryFiltersAndSortBuildTheQuery(t *testing.T) {
	t.Parallel()
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":0,"nodes":[]}}}`))
	}))
	t.Cleanup(srv.Close)
	q := source.Query{Game: "g", Key: "gk", Page: 1, Categories: []string{"Maps", "Audio"}, ExcludeCategories: []string{"Misc"}, Sort: source.SortDownloads}
	if _, err := (Driver{URL: srv.URL}).Search(t.Context(), q); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`op:OR, filter:[`, `{value:\"Maps\"}, {value:\"Misc\", op:NOT_EQUALS}`, `{value:\"Audio\"}`, "downloads:{direction:DESC}"} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("%q missing: %s", want, gotBody)
		}
	}
}

func TestSearchTellsABusySiteFromARefusedRequest(t *testing.T) {
	code := http.StatusTooManyRequests
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
	defer srv.Close()
	d := Driver{URL: srv.URL}
	q := source.Query{Game: "stardew", Key: "stardewvalley", Page: source.FirstPage}
	var busy *source.BusyError
	if _, err := d.Search(t.Context(), q); !errors.As(err, &busy) {
		t.Fatalf("a busy site: %v", err)
	}
	code = http.StatusForbidden
	var status *source.StatusError
	if _, err := d.Search(t.Context(), q); !errors.As(err, &status) || !status.Auth() {
		t.Fatalf("a refused request: %v", err)
	}
	code = http.StatusUnauthorized
	if _, err := d.Search(t.Context(), q); !errors.As(err, &status) || !status.Auth() {
		t.Fatalf("a rejected key: %v", err)
	}
}

func TestSearchAsksForTheQuerysPageSize(t *testing.T) {
	t.Parallel()
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":0,"nodes":[]}}}`))
	}))
	t.Cleanup(srv.Close)
	d := Driver{URL: srv.URL}
	if _, err := d.Search(t.Context(), source.Query{Key: "stardewvalley", Page: 2, Size: d.MaxPageSize()}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, "count: 80, offset: 80") {
		t.Fatalf("body %s", gotBody)
	}
}

func TestSearchGraphQLErrorsSurface(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"bad"}]}`))
	}))
	t.Cleanup(srv.Close)
	_, err := Driver{URL: srv.URL}.Search(t.Context(), source.Query{Key: "stardewvalley", Page: 1})
	if err == nil || !strings.Contains(err.Error(), "bad") {
		t.Fatalf("got %v", err)
	}
}
