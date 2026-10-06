package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
)

// graphQLServer answers legacyModsByDomain for every id it is asked about, counting requests.
func graphQLServer(t *testing.T, status func(n int32) int) (*Client, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		if code := status(n); code != http.StatusOK {
			w.WriteHeader(code)
			return
		}
		var in struct {
			Variables struct {
				IDs []struct {
					Domain string `json:"gameDomain"`
					ModID  int    `json:"modId"`
				} `json:"ids"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Variables.IDs) > 100 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var nodes []string
		for _, id := range in.Variables.IDs {
			nodes = append(nodes, fmt.Sprintf(`{"modId":%d,"name":"Mod %d","version":"1.%d","status":"published","endorsements":%d,"updatedAt":"2026-01-02T03:04:05Z","uploader":{"name":"Ann"},"viewerEndorsed":true}`, id.ModID, id.ModID, id.ModID, id.ModID))
		}
		_, _ = fmt.Fprintf(w, `{"data":{"legacyModsByDomain":{"nodes":[%s]}}}`, strings.Join(nodes, ","))
	}))
	t.Cleanup(srv.Close)
	c := New("test").WithKey("k")
	c.BaseURL = srv.URL
	return c, &requests
}

func TestModsByDomainCostsOneRequestPerHundredMods(t *testing.T) {
	c, requests := graphQLServer(t, func(int32) int { return http.StatusOK })
	ids := make([]int, 800)
	for i := range ids {
		ids[i] = i + 1
	}
	got, err := c.ModsByDomain(context.Background(), "lethalcompany", ids)
	if err != nil || len(got) != 800 {
		t.Fatalf("got %d mods, %v", len(got), err)
	}
	if n := requests.Load(); n > 8 {
		t.Fatalf("800 mods cost %d requests, want at most 8", n)
	}
	p := got[7].Page()
	if p.Name != "Mod 7" || p.Version != "1.7" || !p.Available || p.Author != "Ann" || p.Endorsement != "Endorsed" || p.Updated.Year() != 2026 {
		t.Fatalf("page = %+v", p)
	}
}

func TestModsByDomainStopsAtARateLimitWithoutRetrying(t *testing.T) {
	c, requests := graphQLServer(t, func(n int32) int {
		if n == 2 {
			return http.StatusTooManyRequests
		}
		return http.StatusOK
	})
	ids := make([]int, 450)
	for i := range ids {
		ids[i] = i + 1
	}
	got, err := c.ModsByDomain(context.Background(), "lethalcompany", ids)
	var limit *RateLimitError
	if !errors.As(err, &limit) || len(got) != 100 || requests.Load() != 2 {
		t.Fatalf("got %d mods after %d requests, err %v", len(got), requests.Load(), err)
	}
}

// The answer's shape is Nexus's own (string ids, a requirement of another game, an outside link).
func TestModsByDomainReadsRequirements(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"data":{"legacyModsByDomain":{"nodes":[{"modId":3753,"gameId":"1303","name":"SVE",
			"modRequirements":{"nexusRequirements":{"nodes":[
				{"modId":"1915","modName":"Content Patcher","url":"","externalRequirement":false,"notes":"Framework","gameId":"1303"},
				{"modId":"77","modName":"Other Game Mod","url":"","externalRequirement":false,"notes":"","gameId":"9"},
				{"modId":"","modName":"Some Tool","url":"https://example.org/tool","externalRequirement":true,"notes":"","gameId":""}]}}}]}}}`)
	}))
	t.Cleanup(srv.Close)
	c := New("test").WithKey("k")
	c.BaseURL = srv.URL
	got, err := c.ModsByDomain(context.Background(), "stardewvalley", []int{3753})
	if err != nil {
		t.Fatal(err)
	}
	want := []Requirement{
		{ModID: 1915, Name: "Content Patcher", URL: "https://www.nexusmods.com/stardewvalley/mods/1915", Notes: "Framework"},
		{Name: "Other Game Mod", External: true},
		{Name: "Some Tool", URL: "https://example.org/tool", External: true},
	}
	if r := got[3753].Page().Requirements; !slices.Equal(r, want) {
		t.Fatalf("requirements = %+v", r)
	}
}
