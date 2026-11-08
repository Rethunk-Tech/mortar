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

// A player reads the site's name and status, not the HTTP client's wording.
func TestPageURLSaysTheSitesNameWhenTheProjectIsMissing(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	_, err := Driver{URL: srv.URL, HTTP: srv.Client()}.PageURL(t.Context(), "nope")
	if err == nil || err.Error() != "Modrinth answered 404 Not Found" {
		t.Fatalf("err = %v", err)
	}
}
