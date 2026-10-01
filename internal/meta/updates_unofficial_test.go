package meta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckUpdatesReadsUnofficialFields(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
		  {"id":"me.a","suggestedUpdate":{"version":"2.0.0","url":"https://n.test/a"},
		    "metadata":{"unofficialUpdate":{"version":"2.1.0-unofficial.1-x","url":"https://smapi.io/u"}}},
		  {"id":"me.b","suggestedUpdate":{"version":"1.1.0","url":"https://n.test/b","unofficialForSmapi":{"version":"1.2.0-unofficial","url":"https://smapi.io/b"}},
		    "metadata":{}},
		  {"id":"me.c","metadata":{"unofficial":{"version":"3.0.0-unofficial","url":"https://smapi.io/c"}}}
		]`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), UpdatesURL: srv.URL}
	got := c.CheckUpdates(context.Background(), UpdateRequest{Mods: []InstalledMod{{ID: "me.a"}, {ID: "me.b"}, {ID: "me.c"}}})
	if len(got) != 3 || got[0].Suggested == nil || got[0].Suggested.Version != "2.0.0" ||
		got[0].Unofficial == nil || got[0].Unofficial.Version != "2.1.0-unofficial.1-x" ||
		got[1].Unofficial == nil || got[1].Unofficial.Version != "1.2.0-unofficial" ||
		got[2].Unofficial == nil || got[2].Unofficial.Version != "3.0.0-unofficial" {
		t.Fatalf("got %+v", got)
	}
}
