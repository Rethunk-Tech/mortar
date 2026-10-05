package nexus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestSearchQueryAndPaging(t *testing.T) {
	t.Parallel()
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != graphQL {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":27,"nodes":[{"modId":1348,"name":"SpaceCore","summary":"s","author":"a","version":"1","endorsements":233359,"downloads":1,"pictureUrl":"p","updatedAt":"t","adultContent":true}]}}}`))
	}))
	t.Cleanup(srv.Close)
	page, err := Driver{URL: srv.URL}.Search(context.Background(), source.Query{Game: "stardew", Key: "stardewvalley", Text: "space", Page: 2, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 27 || len(page.Items) != 1 || page.Items[0].ID != "1348" || page.Items[0].Source != "nexus" || !page.Items[0].Adult {
		t.Fatalf("page %+v", page)
	}
	if page.Items[0].URL != "https://www.nexusmods.com/stardewvalley/mods/1348" {
		t.Fatalf("url %s", page.Items[0].URL)
	}
	var body searchBody
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`gameDomainName:[{value:"stardewvalley"}]`,
		`name:[{value:"space", op:WILDCARD}]`,
		"sort:[{endorsements:{direction:DESC}}]",
		"count: 20, offset: 20",
		"adultContent",
	} {
		if !strings.Contains(body.Query, want) {
			t.Fatalf("%q missing: %s", want, body.Query)
		}
	}
}

func TestSearchNeedsTheGameKey(t *testing.T) {
	t.Parallel()
	if _, err := (Driver{}).Search(context.Background(), source.Query{Game: "g", Page: 1}); err == nil {
		t.Fatal("want an error without a Nexus key")
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
	_, err := Driver{URL: srv.URL}.Search(context.Background(), source.Query{Key: "stardewvalley", Text: "a\" }) { x } #\\\x01", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	var body searchBody
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatal(err)
	}
	if want := `op:WILDCARD`; !strings.Contains(body.Query, `{value:"a\" }) { x } #\\\u0001", `+want) {
		t.Fatalf("text not carried as one escaped literal: %s", body.Query)
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
	if _, err := (Driver{URL: srv.URL}).Search(context.Background(), source.Query{Game: "g", Key: "gk", Page: 1}); err != nil {
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
	if _, err := (Driver{URL: srv.URL}).Search(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`op:OR, filter:[`, `{value:\"Maps\"}, {value:\"Misc\", op:NOT_EQUALS}`, `{value:\"Audio\"}`, "downloads:{direction:DESC}"} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("%q missing: %s", want, gotBody)
		}
	}
}
