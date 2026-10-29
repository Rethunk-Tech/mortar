package nexus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCollectionArchiveGivesUpOnAStalledMirror(t *testing.T) {
	old := mirrorTimeout
	mirrorTimeout = 100 * time.Millisecond
	t.Cleanup(func() { mirrorTimeout = old })
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mirror" {
			w.WriteHeader(http.StatusOK)
			_ = http.NewResponseController(w).Flush()
			<-r.Context().Done()
			return
		}
		_, _ = w.Write([]byte(`{"download_links":[{"URI":"` + srv.URL + `/mirror"}]}`))
	}))
	t.Cleanup(srv.Close)
	c := New("9.9")
	c.BaseURL, c.CacheDir, c.HTTP = srv.URL, t.TempDir(), srv.Client()
	c = c.WithKey("k")
	start := time.Now()
	if _, err := c.CollectionArchive(context.Background(), "/v2/collections/x/revisions/1/download_link"); err == nil {
		t.Fatal("stalled mirror succeeded")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("did not give up promptly")
	}
}

func TestCollectionArchiveAllowsASlowSteadyMirror(t *testing.T) {
	old := mirrorTimeout
	mirrorTimeout = 200 * time.Millisecond
	t.Cleanup(func() { mirrorTimeout = old })
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mirror" {
			for range 6 {
				_, _ = w.Write([]byte("x"))
				_ = http.NewResponseController(w).Flush()
				time.Sleep(100 * time.Millisecond)
			}
			return
		}
		_, _ = w.Write([]byte(`{"download_links":[{"URI":"` + srv.URL + `/mirror"}]}`))
	}))
	t.Cleanup(srv.Close)
	c := New("9.9")
	c.BaseURL, c.CacheDir, c.HTTP = srv.URL, t.TempDir(), srv.Client()
	c = c.WithKey("k")
	b, err := c.CollectionArchive(context.Background(), "/v2/collections/x/revisions/1/download_link")
	if err != nil || len(b) != 6 {
		t.Fatalf("got %d bytes, err %v", len(b), err)
	}
}
