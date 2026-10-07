package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	var all []Release
	if err := json.Unmarshal([]byte(releasesJSON), &all); err != nil {
		t.Fatal(err)
	}
	r, assets, err := Select(all, "")
	if err != nil || r.Tag != "v2.1.0" || len(assets) != 3 {
		t.Fatalf("newest = %q %v %v", r.Tag, assets, err)
	}
	r, assets, err = Select(all, "2.0.0")
	if err != nil || r.Tag != "v2.0.0" || len(assets) != 1 {
		t.Fatalf("requested = %q %v %v", r.Tag, assets, err)
	}
	if _, _, err = Select(all, "3.0.0-beta"); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("prerelease must not match: %v", err)
	}
	if _, _, err = Select(all, "1.0.0"); !errors.Is(err, ErrNoArchive) {
		t.Fatalf("no archive: %v", err)
	}
}

func TestReleasesCacheAndStale(t *testing.T) {
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

func TestRateLimit(t *testing.T) {
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
}

func TestDownload(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("data")) }))
	t.Cleanup(srv.Close)
	c := &Client{APIBase: srv.URL, CacheDir: t.TempDir(), HTTP: srv.Client()}
	var last int64
	path, err := c.Download(context.Background(), Asset{Name: "a.zip", URL: c.APIBase + "/dl/a.zip"}, func(done, _ int64) { last = done })
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(path)) }()
	if b, _ := fsx.ReadFile(path); string(b) != "data" || last != 4 || !strings.HasSuffix(path, ".zip") {
		t.Fatalf("file = %q, progress = %d, path = %s", b, last, path)
	}
	over := filepath.Join(t.TempDir(), "over.bin")
	if err := Download(context.Background(), nil, c.APIBase, over, 3, nil); err == nil {
		t.Fatal("body over the cap must fail")
	}
}

func TestDownloadCancelsAStalledBody(t *testing.T) {
	old := downloadIdle
	downloadIdle = 80 * time.Millisecond
	t.Cleanup(func() { downloadIdle = old })
	r, w := io.Pipe()
	t.Cleanup(func() { _ = w.Close() })
	_, err := copyIdle(context.Background(), io.Discard, r, downloadIdle)
	if err == nil {
		t.Fatal("stalled body must fail")
	}
}

func TestDownloadResumesWithRangeWhenTheServerSupportsIt(t *testing.T) {
	const body = "abcdefghij0123456789"
	var sawRange atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=10-" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sawRange.Store(true)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 10-19/%d", len(body)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, body[10:])
	}))
	t.Cleanup(srv.Close)
	dest := filepath.Join(t.TempDir(), "m.zip")
	if err := os.WriteFile(dest, []byte(body[:10]), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Download(context.Background(), srv.Client(), srv.URL, dest, 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(dest)
	if err != nil || string(got) != body || !sawRange.Load() {
		t.Fatalf("got %q range %v err %v", got, sawRange.Load(), err)
	}
}

func TestKey(t *testing.T) {
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
