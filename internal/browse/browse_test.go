package browse

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSearchNexusQueryAndPaging(t *testing.T) {
	t.Parallel()
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != nexusGraphQL {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type %q", r.Header.Get("Content-Type"))
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":27,"nodes":[{"modId":1348,"name":"SpaceCore","summary":"s","author":"a","version":"1","endorsements":233359,"downloads":1,"pictureUrl":"p","updatedAt":"t"}]}}}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{NexusURL: srv.URL, Version: "test"}
	page, err := c.Search(context.Background(), "stardewvalley", "nexus", "space", 2)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 27 || len(page.Items) != 1 || page.Items[0].ID != "1348" || page.Items[0].Name != "SpaceCore" {
		t.Fatalf("page %+v", page)
	}
	if page.Items[0].URL != "https://www.nexusmods.com/stardewvalley/mods/1348" {
		t.Fatalf("url %s", page.Items[0].URL)
	}
	var body nexusSearchBody
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.Query, `gameDomainName:[{value:"stardewvalley"}]`) {
		t.Fatalf("domain missing: %s", body.Query)
	}
	if !strings.Contains(body.Query, `name:[{value:"space", op:WILDCARD}]`) {
		t.Fatalf("wildcard missing: %s", body.Query)
	}
	if !strings.Contains(body.Query, "sort:[{endorsements:{direction:DESC}}]") {
		t.Fatalf("sort missing: %s", body.Query)
	}
	if !strings.Contains(body.Query, "count: 20, offset: 20") {
		t.Fatalf("paging missing: %s", body.Query)
	}
}

func TestSearchNexusMarksInstalled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":1,"nodes":[{"modId":1348,"name":"SpaceCore"},{"modId":99,"name":"Other"}]}}}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{
		NexusURL: srv.URL,
		Installed: func(source, id string) bool {
			return source == sourceNexus && id == "1348"
		},
	}
	page, err := c.Search(context.Background(), "stardewvalley", "nexus", "x", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || !page.Items[0].Installed || page.Items[1].Installed {
		t.Fatalf("installed marking: %+v", page.Items)
	}
}

func TestSearchGitHubQueryPagingAndInstalled(t *testing.T) {
	t.Parallel()
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"total_count":3,"items":[{"full_name":"Pathoschild/SMAPI","description":"d","html_url":"https://github.com/Pathoschild/SMAPI","stargazers_count":9,"updated_at":"u","owner":{"login":"Pathoschild","avatar_url":"a"}}]}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{
		GitHubURL: srv.URL,
		Installed: func(source, id string) bool {
			return source == sourceGitHub && id == "Pathoschild/SMAPI"
		},
	}
	page, err := c.Search(context.Background(), "stardewvalley", "github", "smapi", 2)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Items) != 1 || page.Items[0].Stars != 9 || !page.Items[0].Installed {
		t.Fatalf("page %+v", page)
	}
	if !strings.Contains(gotURL, "q=smapi+topic%3Astardew-valley-mod") && !strings.Contains(gotURL, "topic:stardew-valley-mod") {
		t.Fatalf("query %s", gotURL)
	}
	if !strings.Contains(gotURL, "sort=stars") || !strings.Contains(gotURL, "per_page=20") || !strings.Contains(gotURL, "page=2") {
		t.Fatalf("paging %s", gotURL)
	}
}

func TestSearchGitHubRateLimit(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	c := &Client{GitHubURL: srv.URL}
	_, err := c.Search(context.Background(), "stardewvalley", "github", "x", 1)
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("got %v", err)
	}
}

func TestSearchGitHubCaches(t *testing.T) {
	t.Parallel()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	}))
	t.Cleanup(srv.Close)
	now := time.Unix(1_700_000_000, 0).UTC()
	c := &Client{GitHubURL: srv.URL, Now: func() time.Time { return now }, Cache: &RepoCache{ttl: githubCacheTTL}}
	if _, err := c.Search(context.Background(), "stardewvalley", "github", "x", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(context.Background(), "stardewvalley", "github", "x", 1); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("hits %d", hits)
	}
}
