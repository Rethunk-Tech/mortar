package updatesvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/github"
)

func TestReleaseNotes(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
		case "/repos/Rethunk-AI/mortar/releases/tags/v1.2.0":
			_, _ = w.Write([]byte(`{"body":"  ## Features\n- thing\n"}`))
		case "/repos/Rethunk-AI/mortar/releases/tags/v1.3.0":
			w.Header().Set("X-Ratelimit-Remaining", "0")
			w.WriteHeader(http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	s := &Service{gh: &github.Client{HTTP: srv.Client(), APIBase: srv.URL, CacheDir: t.TempDir()}}
	for _, tc := range []struct {
		name, version string
		want          ReleaseNotes
		calls         int32
	}{
		{"found", "1.2.0", ReleaseNotes{Version: "1.2.0", Notes: "## Features\n- thing", Available: true}, 1},
		{"v prefix, cached", "v1.2.0", ReleaseNotes{Version: "v1.2.0", Notes: "## Features\n- thing", Available: true}, 0},
		{"missing", "9.9.9", ReleaseNotes{Version: "9.9.9"}, 1},
		{"rate limited", "1.3.0", ReleaseNotes{Version: "1.3.0"}, 1},
		{"empty", " ", ReleaseNotes{Version: " "}, 0},
	} {
		hits.Store(0)
		got := s.ReleaseNotes(context.Background(), tc.version)
		if got != tc.want || hits.Load() != tc.calls {
			t.Errorf("%s: %+v, %d calls", tc.name, got, hits.Load())
		}
	}
	srv.Close()
	got := s.ReleaseNotes(context.Background(), "1.4.0")
	if got.Available {
		t.Errorf("offline: %+v", got)
	}
}
