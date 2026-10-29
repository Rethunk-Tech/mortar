package meta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/go-smapiweb"
)

func TestCheckUpdatesAsksAgainAfterAnotherBuildCached(t *testing.T) {
	t.Parallel()
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked.Add(1)
		_, _ = w.Write([]byte(`[{"id":"A.Mod"}]`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), UpdatesURL: srv.URL}
	req := UpdateRequest{Mods: []smapiweb.InstalledMod{{ID: "A.Mod", Version: "1.0.0"}}}
	c.CheckUpdates(context.Background(), req)
	c.CheckUpdates(context.Background(), req)
	if n := asked.Load(); n != 1 {
		t.Fatalf("this build's cache answers the second check: asked %d times", n)
	}
	cache := c.Cache()
	cached, ok := diskcache.Peek[map[string]diskcache.Entry[smapiweb.UpdateResult]](cache, updatesFile)
	if !ok || cached.Stamp == "" {
		t.Fatalf("the cache carries no build stamp: %+v", cached)
	}
	cached.Stamp = "another-build"
	for k, v := range cached.Value {
		v.Stamp = ""
		cached.Value[k] = v
	}
	diskcache.PutEntry(cache, updatesFile, cached)
	c.CheckUpdates(context.Background(), req)
	if n := asked.Load(); n != 2 {
		t.Fatalf("another build's cache is asked again: asked %d times", n)
	}
	cached, _ = diskcache.Peek[map[string]diskcache.Entry[smapiweb.UpdateResult]](cache, updatesFile)
	cached.Stamp = "another-build"
	for k, v := range cached.Value {
		v.Stamp = "another-build"
		cached.Value[k] = v
	}
	other := smapiweb.InstalledMod{ID: "B.Mod", Version: "2.0.0"}
	cached.Value[req.key(other)] = diskcache.Entry[smapiweb.UpdateResult]{Fetched: time.Now(), Value: smapiweb.UpdateResult{ID: "B.Mod", Known: true}, Stamp: "another-build"}
	diskcache.PutEntry(cache, updatesFile, cached)
	c.CheckUpdates(context.Background(), req)
	cached, _ = diskcache.Peek[map[string]diskcache.Entry[smapiweb.UpdateResult]](cache, updatesFile)
	if _, kept := cached.Value[req.key(other)]; !kept {
		t.Fatal("a check of one mod dropped another build's answer for a mod it did not ask about")
	}
}

// An hour-old answer is shown at once while smapi.io is asked again in the background, so a hanging API never holds a
// check up; the background answer replaces it.
func TestCheckUpdatesServesAnOldAnswerAndAsksAgainInTheBackground(t *testing.T) {
	release := make(chan struct{})
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if asked.Add(1) > 1 {
			select {
			case <-release:
			case <-time.After(3 * time.Second):
			}
		}
		_, _ = w.Write([]byte(`[{"id":"A.Mod","metadata":{"compatibilityStatus":"Ok"}}]`))
	}))
	t.Cleanup(srv.Close)
	now := time.Now()
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), UpdatesURL: srv.URL, Now: func() time.Time { return now }}
	req := UpdateRequest{Mods: []smapiweb.InstalledMod{{ID: "A.Mod", Version: "1.0.0"}}}
	c.CheckUpdates(context.Background(), req)
	now = now.Add(2 * time.Hour)
	start := time.Now()
	got := c.CheckUpdates(context.Background(), req)
	if waited := time.Since(start); waited > time.Second || !got[0].Known || got[0].Compatibility != "Ok" {
		t.Fatalf("the old answer waited %v for the API: %+v", waited, got[0])
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for asked.Load() < 2 || c.updatesRefreshing.Load() {
		if time.Now().After(deadline) {
			t.Fatalf("no background refresh: asked %d times", asked.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
