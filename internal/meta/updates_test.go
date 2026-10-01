package meta

import (
	"context"
	"net/http"
	"net/http/httptest"
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
