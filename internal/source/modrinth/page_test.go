package modrinth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageURLNamesTheSlugAndValidIDRefusesPathTricks(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project/AANobbMI" {
			t.Errorf("request %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"slug":"sodium"}`))
	}))
	t.Cleanup(srv.Close)
	got, err := Driver{URL: srv.URL, HTTP: srv.Client()}.PageURL(context.Background(), "AANobbMI")
	if err != nil || got != "https://modrinth.com/project/sodium" {
		t.Fatalf("%q %v", got, err)
	}
	for id, want := range map[string]bool{"AANobbMI": true, "my-mod_2.0": true, "": false, "../x": false, "a/b": false, "a b": false} {
		if ValidID(id) != want {
			t.Errorf("ValidID(%q) = %v", id, !want)
		}
	}
}
