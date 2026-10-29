package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Not parallel: the shown rate is process-wide.
func TestReleaseNotesNotesRate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ratelimit-Limit", "5000")
		w.Header().Set("X-Ratelimit-Remaining", "4321")
		w.Header().Set("X-Ratelimit-Reset", "1700000000")
		_, _ = w.Write([]byte(`{"body":"hi"}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{APIBase: srv.URL, CacheDir: t.TempDir()}
	if _, err := c.ReleaseNotes(context.Background(), "o", "r", "v1"); err != nil {
		t.Fatal(err)
	}
	if r := CurrentRate(); !r.Known || r.Remaining != 4321 {
		t.Fatalf("rate %+v", r)
	}
}
