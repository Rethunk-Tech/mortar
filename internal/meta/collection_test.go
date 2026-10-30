package meta

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestCollectionDropsNonWebLinksFromAnOlderCache(t *testing.T) {
	c := &Client{CacheDir: t.TempDir()}
	diskcache.PutEntry(c.Cache(), "nexus-collection-stardewvalley-slug-0.json", diskcache.Entry[nexusmods.Collection]{Fetched: time.Now(), Value: nexusmods.Collection{External: []nexusmods.CollectionExternal{
		{Name: "app", URL: "steam://run/1"}, {Name: "page", URL: "https://example.com/mod"},
	}}})

	got, err := c.Collection(t.Context(), "stardewvalley", "slug", 0)

	if err != nil || len(got.External) != 2 || got.External[0].URL != "" || got.External[1].URL != "https://example.com/mod" {
		t.Fatalf("external = %+v, %v", got.External, err)
	}
}

// Collections are read without an account, as Mortar, and a repeat inside the TTL is served from the cache.
func TestCollectionReadsAnonymouslyAndCaches(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Apikey") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("Application-Name") != "Mortar" || r.Header.Get("Application-Version") != "9.9" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"data":{"collectionRevision":{"revisionNumber":4,"collection":{"name":"Cozy Farm","slug":"cozy-farm","user":{"name":"Pat"}},"modFiles":[{"fileId":11,"file":{"modId":100}}]}}}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{CacheDir: t.TempDir(), NexusURL: srv.URL, Version: "9.9"}
	for range 2 {
		got, err := c.Collection(t.Context(), "stardewvalley", "cozy-farm", 0)
		if err != nil || got.Name != "Cozy Farm" || len(got.Files) != 1 {
			t.Fatalf("collection = %+v, %v", got, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d, want one", hits.Load())
	}
}

// A collection Nexus does not know is a not-found the player reads in Mortar's words.
func TestCollectionNotFoundKeepsItsWording(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"collectionRevision":null}}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{CacheDir: t.TempDir(), NexusURL: srv.URL}
	_, err := c.Collection(t.Context(), "stardewvalley", "missing", 0)
	if usererr.KindOf(err) != usererr.NotFound || !strings.Contains(err.Error(), `nexus has no collection "missing"`) {
		t.Fatalf("error = %v", err)
	}
}
