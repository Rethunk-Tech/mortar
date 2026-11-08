package modrinth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageURLNamesTheSlug(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project/AANobbMI" {
			t.Errorf("request %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"slug":"sodium"}`))
	}))
	t.Cleanup(srv.Close)
	got, err := Driver{URL: srv.URL, HTTP: srv.Client()}.PageURL(t.Context(), "AANobbMI")
	if err != nil || got != "https://modrinth.com/project/sodium" {
		t.Fatalf("%q %v", got, err)
	}
}
