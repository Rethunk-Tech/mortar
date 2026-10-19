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

const resumeBody = "abcdefghij0123456789"

// rangeServer honours Range from the start of body (a validator mismatch in If-Range answers the whole body, as
// HTTP says); seen collects each request's Range and If-Range.
func rangeServer(t *testing.T, body, etag string, seen *[]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = append(*seen, r.Header.Get("Range")+"|"+r.Header.Get("If-Range"))
		w.Header().Set("ETag", etag)
		var from int
		_, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-", &from)
		if err != nil || (r.Header.Get("If-Range") != "" && r.Header.Get("If-Range") != etag) {
			_, _ = io.WriteString(w, body)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", from, len(body)-1, len(body)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, body[from:])
	}))
	t.Cleanup(srv.Close)
	return srv
}

func seedPartial(t *testing.T, dest, partial string, m resumeMeta) {
	t.Helper()
	if err := os.WriteFile(dest, []byte(partial), 0o600); err != nil {
		t.Fatal(err)
	}
	saveResume(dest, m)
}

func TestDownloadResumesFromThePartialWithIfRange(t *testing.T) {
	var seen []string
	srv := rangeServer(t, resumeBody, `"v1"`, &seen)
	dest := filepath.Join(t.TempDir(), "m.zip")
	seedPartial(t, dest, resumeBody[:10], resumeMeta{URL: srv.URL, ETag: `"v1"`, ExpectedSize: int64(len(resumeBody))})
	if err := Download(context.Background(), srv.Client(), srv.URL, dest, 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := fsx.ReadFile(dest)
	if string(got) != resumeBody || len(seen) != 1 || seen[0] != `bytes=10-|"v1"` {
		t.Fatalf("got %q requests %v", got, seen)
	}
}

func TestDownloadStartsOverWhenTheValidatorChanged(t *testing.T) {
	var seen []string
	srv := rangeServer(t, resumeBody, `"v2"`, &seen)
	dest := filepath.Join(t.TempDir(), "m.zip")
	seedPartial(t, dest, "STALEPARTI", resumeMeta{URL: srv.URL, ETag: `"v1"`, ExpectedSize: int64(len(resumeBody))})
	if err := Download(context.Background(), srv.Client(), srv.URL, dest, 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.ReadFile(dest); string(got) != resumeBody {
		t.Fatalf("got %q", got)
	}
}

func TestDownloadStartsOverWhenTheServerIgnoresRange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, resumeBody) }))
	t.Cleanup(srv.Close)
	dest := filepath.Join(t.TempDir(), "m.zip")
	seedPartial(t, dest, "STALEPARTI", resumeMeta{URL: srv.URL, ExpectedSize: int64(len(resumeBody)), LastModified: "x"})
	if err := Download(context.Background(), srv.Client(), srv.URL, dest, 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.ReadFile(dest); string(got) != resumeBody {
		t.Fatalf("got %q", got)
	}
}

func TestDownloadResumesOnANewAddressOfTheSameSize(t *testing.T) {
	var seen []string
	srv := rangeServer(t, resumeBody, `"other-cdn"`, &seen)
	dest := filepath.Join(t.TempDir(), "m.zip")
	seedPartial(t, dest, resumeBody[:10], resumeMeta{URL: "https://old.example/f", ETag: `"v1"`, ExpectedSize: int64(len(resumeBody))})
	if err := Download(context.Background(), srv.Client(), srv.URL, dest, 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.ReadFile(dest); string(got) != resumeBody || len(seen) != 1 || seen[0] != "bytes=10-|" {
		t.Fatalf("got %q requests %v", got, seen)
	}
}

func TestDownloadStartsOverWhenTheTotalChanged(t *testing.T) {
	var seen []string
	srv := rangeServer(t, resumeBody+"MORE", `"v1"`, &seen)
	dest := filepath.Join(t.TempDir(), "m.zip")
	seedPartial(t, dest, resumeBody[:10], resumeMeta{URL: "https://old.example/f", ExpectedSize: int64(len(resumeBody))})
	if err := Download(context.Background(), srv.Client(), srv.URL, dest, 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.ReadFile(dest); string(got) != resumeBody+"MORE" {
		t.Fatalf("got %q", got)
	}
}

func TestDownloadDropsAPartialNothingVouchesFor(t *testing.T) {
	var seen []string
	srv := rangeServer(t, resumeBody, `"v1"`, &seen)
	dest := filepath.Join(t.TempDir(), "m.zip")
	if err := os.WriteFile(dest, []byte("unknown"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Download(context.Background(), srv.Client(), srv.URL, dest, 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.ReadFile(dest); string(got) != resumeBody || seen[0] != "|" {
		t.Fatalf("got %q requests %v", got, seen)
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

func TestBucketSharesTheRateAcrossCallers(t *testing.T) {
	var b bucket
	now := time.Unix(1_000, 0)
	const rate = 100 << 10
	if w := b.reserve(rate/4, rate, now); w != 0 {
		t.Fatalf("the burst waited %v", w)
	}
	// A second download draws on the same, now empty, bucket.
	if w := b.reserve(rate/4, rate, now); w != 250*time.Millisecond {
		t.Fatalf("second caller waits %v, want 250ms", w)
	}
	// Raising the limit at once shortens the wait for the same bytes.
	if w := b.reserve(rate/4, 2*rate, now); w >= 250*time.Millisecond*2 {
		t.Fatalf("a raised limit still waits %v", w)
	}
}

func TestDownloadHonoursTheSpeedLimit(t *testing.T) {
	body := strings.Repeat("x", 120<<10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) }))
	t.Cleanup(srv.Close)
	SpeedLimit = func() int64 { return 200 << 10 }
	t.Cleanup(func() { SpeedLimit, shared = nil, bucket{} })
	dest := filepath.Join(t.TempDir(), "m.zip")
	start := time.Now()
	if err := Download(context.Background(), srv.Client(), srv.URL, dest, 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	// 120 KiB at 200 KiB/s with a 50 KiB burst leaves 70 KiB, 0.35 s.
	if took := time.Since(start); took < 250*time.Millisecond {
		t.Fatalf("120 KiB arrived in %v under a 200 KiB/s limit", took)
	}
	if got, _ := fsx.ReadFile(dest); string(got) != body {
		t.Fatal("limited download is corrupt")
	}
}
