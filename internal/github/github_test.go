package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-httpkit/download"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

const releasesJSON = `[
 {"tag_name":"v3.0.0-beta","prerelease":true,"assets":[{"name":"m.zip","browser_download_url":"x"}]},
 {"tag_name":"v2.9.0","draft":true,"assets":[{"name":"m.zip","browser_download_url":"x"}]},
 {"tag_name":"v2.1.0","assets":[{"name":"Mod-2.1.0.zip","content_type":"application/zip","size":4,"browser_download_url":"URL/dl/a.zip"},
   {"name":"Mod-2.1.0-alt.7z","content_type":"application/x-7z-compressed"},{"name":"src.tar.gz"},{"name":"fake.zip","content_type":"text/html"}]},
 {"tag_name":"v2.0.0","assets":[{"name":"Mod.zip"}]},
 {"tag_name":"v1.0.0","assets":[{"name":"notes.txt"}]}
]`

func server(t *testing.T, hits *atomic.Int32, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return &Client{APIBase: srv.URL, CacheDir: t.TempDir()}
}

func serve(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(releasesJSON)) }

func TestSelect(t *testing.T) {
	t.Parallel()
	var all []Release
	if err := json.Unmarshal([]byte(releasesJSON), &all); err != nil {
		t.Fatal(err)
	}
	r, assets, err := Select(all, "", false)
	if err != nil || r.Tag != "v2.1.0" || len(assets) != 3 {
		t.Fatalf("newest = %q %v %v", r.Tag, assets, err)
	}
	r, assets, err = Select(all, "2.0.0", false)
	if err != nil || r.Tag != "v2.0.0" || len(assets) != 1 {
		t.Fatalf("requested = %q %v %v", r.Tag, assets, err)
	}
	if r, _, err = Select(all, "", true); err != nil || r.Tag != "v3.0.0-beta" {
		t.Fatalf("prereleases asked for = %q %v", r.Tag, err)
	}
	if r, _, err = Select(all, "3.0.0-beta", false); err != nil || r.Tag != "v3.0.0-beta" {
		t.Fatalf("a named prerelease = %q %v", r.Tag, err)
	}
	if _, _, err = Select(all, "1.0.0", false); !errors.Is(err, ErrNoArchive) {
		t.Fatalf("no archive: %v", err)
	}
}

func TestReleasesCacheAndStale(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	fail := false
	c := server(t, &hits, func(w http.ResponseWriter, r *http.Request) {
		if fail {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		if r.URL.Path != "/repos/o/r/releases" {
			t.Errorf("path = %s", r.URL.Path)
		}
		serve(w, r)
	})
	now := time.Now().UTC()
	c.Now = func() time.Time { return now }
	if _, err := c.Releases(context.Background(), "o", "r"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Releases(context.Background(), "o", "r"); err != nil || hits.Load() != 1 {
		t.Fatalf("cache hit: hits = %d, err = %v", hits.Load(), err)
	}
	now = now.Add(2 * time.Hour)
	fail = true
	got, err := c.Releases(context.Background(), "o", "r")
	if err != nil || len(got) != 5 || hits.Load() != 2 {
		t.Fatalf("stale fallback = %d releases, hits %d, err %v", len(got), hits.Load(), err)
	}
	if _, err := (&Client{APIBase: c.APIBase, CacheDir: t.TempDir()}).Releases(context.Background(), "o", "r"); err == nil {
		t.Fatal("no cache and a failing API must error")
	}
}

func TestCachedReleasesNeverAsksAndOldFormatsAreFetchedAgain(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	c := server(t, &hits, serve)
	if _, _, ok := c.CachedReleases("o", "r"); ok {
		t.Fatal("an empty cache answered")
	}
	if _, err := c.Releases(context.Background(), "o", "r"); err != nil {
		t.Fatal(err)
	}
	got, whole, ok := c.CachedReleases("o", "r")
	if !ok || !whole || len(got) != 5 || hits.Load() != 1 {
		t.Fatalf("cached = %d releases, whole %v, ok %v, hits %d", len(got), whole, ok, hits.Load())
	}
	// A file stamped with an older format lacks the asset counts, so the list is read again however young.
	path := filepath.Join(c.CacheDir, releasesName("o", "r"))
	if err := os.WriteFile(path, []byte(`{"fetched":"`+time.Now().UTC().Format(time.RFC3339)+`","value":[{"tag_name":"v0"}],"build":"1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := c.CachedReleases("o", "r"); ok {
		t.Fatal("an old format was read")
	}
	if got, err := c.Releases(context.Background(), "o", "r"); err != nil || len(got) != 5 || hits.Load() != 2 {
		t.Fatalf("old format: %d releases, hits %d, err %v", len(got), hits.Load(), err)
	}
	c.Now = func() time.Time { return time.Now().UTC().Add(-24 * time.Hour) }
	if _, err := c.Releases(context.Background(), "o", "r"); err != nil || hits.Load() != 3 {
		t.Fatalf("an answer stamped after a clock set back was served: hits %d, err %v", hits.Load(), err)
	}
}

// The unversioned names and shapes of earlier builds must never answer: a nil list would stand in for a failed fetch.
func TestEarlierCacheShapesReadAsAbsent(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	c := server(t, &hits, serve)
	now := time.Now().UTC().Format(time.RFC3339)
	key := Key("o", "r", "", "")
	for name, body := range map[string]string{
		key + "-releases.json":                     `{"fetched":"` + now + `","releases":[{"tag_name":"v0"}]}`,
		releasesName("o", "r"):                     `{"fetched":"` + now + `","releases":[{"tag_name":"v0"}]}`,
		Key("o", "r", "v1", "") + "-notes.json":    `{"notes":"old"}`,
		Key("o", "r", "v2", "") + "-notes-v2.json": `{"notes":"old"}`,
	} {
		if err := os.WriteFile(filepath.Join(c.CacheDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, ok := c.CachedReleases("o", "r"); ok {
		t.Fatal("a release list without a value answered")
	}
	if got, err := c.Releases(context.Background(), "o", "r"); err != nil || len(got) != 5 || hits.Load() != 1 {
		t.Fatalf("releases: %d, hits %d, err %v", len(got), hits.Load(), err)
	}
	if notes, ok := diskcache.Peek[string](c.cache(), Key("o", "r", "v2", "")+"-notes-v2.json"); ok {
		t.Fatalf("notes without a value answered %q", notes.Value)
	}
	if notes, ok := diskcache.Peek[string](c.cache(), Key("o", "r", "v1", "")+"-notes-v2.json"); ok {
		t.Fatalf("an unversioned notes file answered %q", notes.Value)
	}
}

func TestRateLimit(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	c := server(t, &hits, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ratelimit-Remaining", "0")
		w.Header().Set("X-Ratelimit-Reset", "1790000000")
		http.Error(w, `{"message":"API rate limit exceeded"}`, http.StatusForbidden)
	})
	_, err := c.Releases(context.Background(), "o", "r")
	var rl *RateLimitError
	if !errors.As(err, &rl) || !rl.Reset.Equal(time.Unix(1790000000, 0)) || !strings.Contains(err.Error(), "try again after") {
		t.Fatalf("err = %v", err)
	}
	// A file stamped with an older format still stands in when the lookup fails.
	if err := os.WriteFile(filepath.Join(c.CacheDir, releasesName("o", "r")), []byte(`{"fetched":"2025-01-01T00:00:00Z","value":[{"tag_name":"v0"}],"build":"1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Releases(context.Background(), "o", "r"); err != nil || len(got) != 1 {
		t.Fatalf("old format under a rate limit: %d releases, err %v", len(got), err)
	}
}

func TestDownloadRefusesABodyOverTheCap(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("data")) }))
	t.Cleanup(srv.Close)
	over := filepath.Join(t.TempDir(), "over.bin")
	err := Download(context.Background(), srv.Client(), srv.URL, over, 3, nil)
	if err == nil || !strings.Contains(err.Error(), "larger than 3 B") || !errors.Is(err, safefs.ErrTooLarge) {
		t.Fatalf("body over the cap: err = %v", err)
	}
}

func TestDownloadReportsGitHubsLimitBeforeTheStatus(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ratelimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	var limit *RateLimitError
	err := Download(context.Background(), srv.Client(), srv.URL, filepath.Join(t.TempDir(), "m.zip"), 1<<20, nil)
	if !errors.As(err, &limit) || errors.Is(err, download.ErrLinkExpired) {
		t.Fatalf("a 403 with no calls left: err = %v", err)
	}
}

func TestDownloadNamesMortarWhenNoRangeResumes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
	}))
	t.Cleanup(srv.Close)
	err := Download(context.Background(), srv.Client(), srv.URL, filepath.Join(t.TempDir(), "m.zip"), 1<<20, nil)
	if err == nil || err.Error() != "the download server sent a range Mortar could not resume" {
		t.Fatalf("err = %v", err)
	}
}

func TestKey(t *testing.T) {
	t.Parallel()
	k := Key("Pathoschild", "StardewMods", "v2.1.0", "Mod (Final) 2.1.0.zip")
	if !strings.HasPrefix(k, "github-pathoschild-stardewmods-v2.1.0-mod-final-2.1.0.zip-") {
		t.Fatalf("key = %q", k)
	}
	if Key("a-b", "c", "t", "x.zip") == Key("a", "b-c", "t", "x.zip") {
		t.Fatal("owner/repo split folded into one key")
	}
	if Key("Me", "Mod", "t", "x.zip") != Key("me", "mod", "t", "x.zip") {
		t.Fatal("GitHub names are case-insensitive but keyed apart")
	}
	if long := Key(strings.Repeat("o", 39), strings.Repeat("r", 100), strings.Repeat("t", 100), strings.Repeat("a", 200)); len(long) > 128 {
		t.Fatalf("key is %d bytes", len(long))
	}
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`).MatchString(Key("_a", "b_", "", "?")) {
		t.Fatal("key breaks the store's rules")
	}
}

func TestVerify(t *testing.T) {
	t.Parallel()
	answer := `[{"id":"Some.Mod","metadata":{"gitHubRepo":"Owner/Repo"}}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if answer == "" {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(answer))
	}))
	defer srv.Close()
	m := func() *meta.Client { return &meta.Client{UpdatesURL: srv.URL, CacheDir: t.TempDir()} }
	ctx := context.Background()
	if ok, err := Verify(ctx, m(), "Some.Mod", "owner", "repo"); !ok || err != nil {
		t.Fatalf("match = %v, %v", ok, err)
	}
	if ok, err := Verify(ctx, m(), "Some.Mod", "other", "repo"); ok || err != nil {
		t.Fatalf("mismatch = %v, %v", ok, err)
	}
	answer = ""
	if ok, err := Verify(ctx, m(), "Some.Mod", "owner", "repo"); ok || !errors.Is(err, ErrUnknown) {
		t.Fatalf("unknown = %v, %v", ok, err)
	}
}

func TestReleasesBetween(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	c := server(t, &hits, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
 {"tag_name":"v2.2.0","body":"too new","published_at":"2026-03-01T00:00:00Z"},
 {"tag_name":"v2.1.0","body":"- fix crash\n- see https://x.test","published_at":"2026-02-01T10:00:00Z"},
 {"tag_name":"v2.0.5","draft":true,"body":"draft"},
 {"tag_name":"nightly","body":"not a version"},
 {"tag_name":"v2.0.0","body":"rewrite","published_at":"2026-01-01T00:00:00Z"},
 {"tag_name":"v1.0.0","body":"installed"}]`))
	})
	got, err := c.ReleasesBetween(context.Background(), "o", "r", "1.0.0", "2.1.0")
	if err != nil || len(got) != 2 || got[0].Tag != "v2.1.0" || got[1].Tag != "v2.0.0" || got[0].Published != "2026-02-01T10:00:00Z" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestRateLimitedPrefersRetryAfter(t *testing.T) {
	t.Parallel()
	var e *RateLimitError
	h := http.Header{
		"X-Ratelimit-Remaining": {"4000"}, "Retry-After": {"60"},
		"X-Ratelimit-Reset": {strconv.FormatInt(time.Now().Add(50*time.Minute).Unix(), 10)},
	}
	if !errors.As(RateLimited(&http.Response{StatusCode: http.StatusForbidden, Header: h}), &e) {
		t.Fatal("not a rate limit")
	}
	if d := time.Until(e.Reset); d < 50*time.Second || d > 60*time.Second {
		t.Fatalf("wait %v", d)
	}
	at := time.Now().Add(90 * time.Second).UTC().Truncate(time.Second)
	h = http.Header{"Retry-After": {at.Format(http.TimeFormat)}}
	if !errors.As(RateLimited(&http.Response{StatusCode: http.StatusTooManyRequests, Header: h}), &e) || !e.Reset.Equal(at) {
		t.Fatalf("date reset %v", e.Reset)
	}
}
