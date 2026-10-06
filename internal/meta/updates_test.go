package meta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
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
