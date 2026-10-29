package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/netstate"
)

func server(t *testing.T, hits *atomic.Int32, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return &Client{APIBase: srv.URL, CacheDir: t.TempDir()}
}

func TestClientWordsGitHubsAnswers(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	status, limited := http.StatusBadGateway, false
	c := server(t, &hits, func(w http.ResponseWriter, _ *http.Request) {
		if limited {
			w.Header().Set("X-Ratelimit-Remaining", "0")
			w.Header().Set("X-Ratelimit-Reset", "1790000000")
		}
		http.Error(w, "x", status)
	})
	ctx := context.Background()

	_, err := c.Releases(ctx, "o", "r")
	if err == nil || err.Error() != "GitHub's releases API answered 502 Bad Gateway" {
		t.Fatalf("releases: %v", err)
	}
	status = http.StatusNotFound
	if _, err = c.ReleasesBetween(ctx, "o", "r", "1", ""); err == nil || err.Error() != "GitHub's releases API answered 404 Not Found" {
		t.Fatalf("between: %v", err)
	}
	if _, err = c.ReleaseNotes(ctx, "o", "r", "v1"); err == nil || err.Error() != "GitHub's release API answered 404 Not Found" {
		t.Fatalf("notes: %v", err)
	}

	status, limited = http.StatusForbidden, true
	for name, call := range map[string]func() error{
		"releases": func() error { _, err := c.Releases(ctx, "o", "r"); return err },
		"notes":    func() error { _, err := c.ReleaseNotes(ctx, "o", "r", "v1"); return err },
	} {
		err := call()
		var rl *RateLimitError
		if !errors.As(err, &rl) || !rl.Reset.Equal(time.Unix(1790000000, 0)) || !strings.Contains(err.Error(), "try again after") {
			t.Fatalf("%s limit: %v", name, err)
		}
	}
}

func TestClientSaysWhenGitHubCannotBeReached(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	c := &Client{APIBase: url, CacheDir: t.TempDir()}
	for name, err := range map[string]error{
		"releases": func() error { _, err := c.Releases(context.Background(), "o", "r"); return err }(),
		"notes":    func() error { _, err := c.ReleaseNotes(context.Background(), "o", "r", "v1"); return err }(),
		"fetch":    func() error { _, err := FetchReleases(context.Background(), nil, url); return err }(),
	} {
		if err == nil || !strings.HasPrefix(err.Error(), "could not reach GitHub: ") {
			t.Fatalf("%s: %v", name, err)
		}
	}
	var unreachable bool
	for _, st := range (&netstate.Service{}).States() {
		unreachable = unreachable || (st.ID == "github" && st.Unreachable)
	}
	if !unreachable {
		t.Fatal("a refused connection did not mark GitHub unreachable")
	}
}

// Not parallel: reachability is process-wide.
func TestClientRecordsReachabilityOfNotesToo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"body":"hi"}`)) }))
	t.Cleanup(srv.Close)
	netstate.Record("github", context.DeadlineExceeded)
	c := &Client{APIBase: srv.URL, CacheDir: t.TempDir()}
	if _, err := c.ReleaseNotes(context.Background(), "o", "r", "v1"); err != nil {
		t.Fatal(err)
	}
	for _, st := range (&netstate.Service{}).States() {
		if st.ID == "github" && st.Unreachable {
			t.Fatal("an answered notes request left GitHub unreachable")
		}
	}
}

func TestZeroClientCachesUnderTheDataFolder(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`[{"tag_name":"v1"}]`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{APIBase: srv.URL}
	for range 2 {
		if _, err := c.Releases(context.Background(), "o", "r"); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d, want the second call served from the cache", hits.Load())
	}
	files, _ := filepath.Glob(filepath.Join(os.Getenv("XDG_DATA_HOME"), "*", "cache", "*-releases-v2.json"))
	if _, _, ok := c.CachedReleases("o", "r"); !ok || len(files) == 0 {
		t.Fatalf("no cache file under the data folder (%v)", files)
	}
}

func TestSelectNamesWhatIsMissing(t *testing.T) {
	t.Parallel()
	all := []Release{{Tag: "V1.0.0", Assets: []Asset{{Name: "notes.txt"}}}}
	if _, _, err := Select(all, "v1.0.0", false); !errors.Is(err, ErrNoArchive) {
		t.Fatalf("a release without an archive: %v", err)
	}
	if _, _, err := Select(all, "2.0.0", false); !errors.Is(err, ErrNoRelease) || err.Error() != "GitHub lists no matching stable release" {
		t.Fatalf("no such release: %v", err)
	}
}
