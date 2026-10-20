package meta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheckUpdatesReadsCompatibilitySummary(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
		  {"id":"Old.Mod","metadata":{
		    "compatibilityStatus":"Abandoned",
		    "compatibilitySummary":"use [New Mod](https://www.nexusmods.com/stardewvalley/mods/5098) instead.",
		    "brokeIn":""
		  }}
		]`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), UpdatesURL: srv.URL}
	got := c.CheckUpdates(context.Background(), UpdateRequest{Mods: []InstalledMod{{ID: "Old.Mod"}}})
	if len(got) != 1 || !got[0].Known || got[0].Compatibility != "Abandoned" ||
		got[0].CompatibilitySummary != "use [New Mod](https://www.nexusmods.com/stardewvalley/mods/5098) instead." {
		t.Fatalf("got %+v", got[0])
	}
}

func TestCheckUpdatesAsksAgainAfterAnotherBuildCached(t *testing.T) {
	t.Parallel()
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked.Add(1)
		_, _ = w.Write([]byte(`[{"id":"A.Mod"}]`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), UpdatesURL: srv.URL}
	req := UpdateRequest{Mods: []InstalledMod{{ID: "A.Mod", Version: "1.0.0"}}}
	c.CheckUpdates(context.Background(), req)
	c.CheckUpdates(context.Background(), req)
	if n := asked.Load(); n != 1 {
		t.Fatalf("this build's cache answers the second check: asked %d times", n)
	}
	path, err := c.cachePath(updatesFile)
	if err != nil {
		t.Fatal(err)
	}
	cached, ok := readEntry[map[string]entry[UpdateResult]](path)
	if !ok || cached.Build == "" {
		t.Fatalf("the cache carries no build stamp: %+v", cached)
	}
	cached.Build = "another-build"
	writeEntry(path, cached)
	c.CheckUpdates(context.Background(), req)
	if n := asked.Load(); n != 2 {
		t.Fatalf("another build's cache is asked again: asked %d times", n)
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
	req := UpdateRequest{Mods: []InstalledMod{{ID: "A.Mod", Version: "1.0.0"}}}
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
