package nexussvc

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/mortar/internal/github"
)

func TestReleaseChangelogsSkipsDraftsAndTrimsTags(t *testing.T) {
	got := releaseChangelogs([]github.Release{
		{Tag: "v2.0.0", Body: " big\n", Published: "2026-01-02T03:04:05Z"},
		{Tag: "v1.9.0", Draft: true},
		{Tag: "1.0.0"},
	})
	if len(got) != 2 || got[0].Version != "2.0.0" || got[0].Date != "2026-01-02" || got[0].Body != "big" || got[1].Version != "1.0.0" {
		t.Fatalf("got %+v", got)
	}
}

// UpdateChangelog and ReleaseChangelog are bound to the frontend, so githubRepo is untrusted: a name that would move
// the API URL's path is refused before any request.
func TestGitHubChangelogsRefuseANameThatMovesTheURL(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	t.Cleanup(srv.Close)
	s := &Service{GitHub: &github.Client{APIBase: srv.URL, CacheDir: t.TempDir()}}
	for _, repo := range []string{"me/..", "o/r?x", "../x", "o/r/s", "o"} {
		if _, err := s.ReleaseChangelog(t.Context(), repo); !errors.Is(err, gh.ErrInvalidRepo) {
			t.Errorf("ReleaseChangelog(%q): %v", repo, err)
		}
		if _, err := s.UpdateChangelog(t.Context(), "g", 0, repo, "1.0.0", "2.0.0"); !errors.Is(err, gh.ErrInvalidRepo) {
			t.Errorf("UpdateChangelog(%q): %v", repo, err)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("%d requests reached the server", hits.Load())
	}
}

func TestCachedReleaseChangelogReadsOnlyWhatIsStored(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`[{"tag_name":"v2.0.0","body":"big","published_at":"2026-01-02T03:04:05Z"}]`))
	}))
	t.Cleanup(srv.Close)
	s := &Service{GitHub: &github.Client{APIBase: srv.URL, CacheDir: t.TempDir()}}
	if got := s.CachedReleaseChangelog("o/r"); got != nil {
		t.Fatalf("before any read: %+v", got)
	}
	if _, err := s.ReleaseChangelog(t.Context(), "o/r"); err != nil {
		t.Fatal(err)
	}
	served := hits.Load()
	got := s.CachedReleaseChangelog("o/r")
	if len(got) != 1 || got[0].Version != "2.0.0" || got[0].Body != "big" {
		t.Fatalf("after a read: %+v", got)
	}
	if s.CachedReleaseChangelog("o/..") != nil || hits.Load() != served {
		t.Fatal("a name that moves the path must read nothing, and no read may reach the server")
	}
}
