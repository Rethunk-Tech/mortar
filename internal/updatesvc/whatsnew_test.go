package updatesvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/github"
)

func TestWhatsNewOfflineKeepsLastRun(t *testing.T) {
	dir := t.TempDir()
	if err := writeLastRun(dir, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "offline", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	s := &Service{info: Info{Version: "1.1.0"}, dir: dir, gh: &github.Client{HTTP: srv.Client(), APIBase: srv.URL, CacheDir: t.TempDir()}}
	got, err := s.WhatsNew(context.Background())
	if err != nil || got.Version != "" {
		t.Fatalf("offline skip: %+v, %v", got, err)
	}
	last, err := readLastRun(dir)
	if err != nil || last != "1.0.0" {
		t.Fatalf("last run kept: %q, %v", last, err)
	}
}

func TestWhatsNewAfterSkippedVersionsListsEachRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/Rethunk-Tech/mortar/releases/tags/v1.2.0":
			_, _ = w.Write([]byte(`{"body":"two"}`))
		case "/repos/Rethunk-Tech/mortar/releases":
			_, _ = w.Write([]byte(`[{"tag_name":"v1.3.0","body":"three"},{"tag_name":"v1.2.0","body":"two"},` +
				`{"tag_name":"v1.1.1","body":"one one"},{"tag_name":"v1.1.0","body":""},{"tag_name":"v1.0.0","body":"zero"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	if err := writeLastRun(dir, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	s := &Service{info: Info{Version: "1.2.0"}, dir: dir, gh: &github.Client{HTTP: srv.Client(), APIBase: srv.URL, CacheDir: t.TempDir()}}
	got, err := s.WhatsNew(context.Background())
	if err != nil || got.Notes != "two" || len(got.Earlier) != 1 || got.Earlier[0] != (ReleaseNotes{Version: "1.1.1", Notes: "one one", Available: true}) {
		t.Fatalf("what's new = %+v, %v", got, err)
	}
}
