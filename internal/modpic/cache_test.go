package modpic

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var png1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
	0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
	0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func pictureURL(host, path string) string {
	return "https://" + host + path
}

func tlsClient(srv *httptest.Server, host string) *http.Client {
	inner := srv.Client()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r2 := r.Clone(r.Context())
		r2.URL.Scheme = "https"
		r2.URL.Host = strings.TrimPrefix(srv.URL, "https://")
		r2.Host = host
		return inner.Transport.RoundTrip(r2)
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetCachesOnce(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png1x1)
	}))
	t.Cleanup(srv.Close)
	host := "staticdelivery.nexusmods.com"
	c := New(t.TempDir(), tlsClient(srv, host))
	u := pictureURL(host, "/mods/1.png")
	b, typ, err := c.get(t.Context(), u)
	if err != nil || typ != "image/png" || len(b) != len(png1x1) {
		t.Fatalf("get: %s %v %d", typ, err, len(b))
	}
	if _, _, err := c.get(t.Context(), u); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("fetched %d times", hits.Load())
	}
}

func TestFetchRejectsNonImage(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "not an image")
	}))
	t.Cleanup(srv.Close)
	host := "staticdelivery.nexusmods.com"
	c := New(t.TempDir(), tlsClient(srv, host))
	_, _, err := c.get(t.Context(), pictureURL(host, "/mods/1.bin"))
	if err == nil {
		t.Fatal("accepted a non-image body")
	}
}

func TestFetchRejectsOversized(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte{0xff, 0xd8, 0xff})
		_, _ = w.Write(make([]byte, MaxSize))
	}))
	t.Cleanup(srv.Close)
	host := "images.nexusmods.com"
	c := New(t.TempDir(), tlsClient(srv, host))
	_, _, err := c.get(t.Context(), pictureURL(host, "/big.jpg"))
	if err == nil {
		t.Fatal("accepted an oversized body")
	}
}

func TestFetchRejectsDisallowedHost(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("fetched a disallowed host")
	}))
	t.Cleanup(srv.Close)
	c := New(t.TempDir(), srv.Client())
	_, _, err := c.get(t.Context(), "https://example.com/x.png")
	if err == nil {
		t.Fatal("accepted a disallowed host")
	}
}

func TestMiddlewarePathTraversal(t *testing.T) {
	h := Middleware(func() *Cache { return New(t.TempDir(), http.DefaultClient) })(http.NotFoundHandler())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://mortar/mod-picture/foo", nil)
	req.URL.Path = "/mod-picture/foo/../../../etc/passwd"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestMiddlewareServesCached(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(png1x1)
	}))
	t.Cleanup(srv.Close)
	host := "staticdelivery.nexusmods.com"
	c := New(t.TempDir(), tlsClient(srv, host))
	u := pictureURL(host, "/mods/1.png")
	h := Middleware(func() *Cache { return c })(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("fell through")
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://mortar"+AssetURL(u), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("code %d type %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec.Body.Len() != len(png1x1) {
		t.Fatalf("body %d", rec.Body.Len())
	}
}

func TestAssetURLEmptyWhenInvalid(t *testing.T) {
	if AssetURL("http://staticdelivery.nexusmods.com/x.png") != "" {
		t.Fatal("http")
	}
	if AssetURL("https://evil.example/x.png") != "" {
		t.Fatal("host")
	}
}

func TestAssetURLTakesEveryBrowseSourcesPictureHost(t *testing.T) {
	for _, host := range []string{"cdn.modrinth.com", "media.forgecdn.net", "avatars.githubusercontent.com", "img.itch.zone"} {
		if AssetURL(pictureURL(host, "/a.png")) == "" {
			t.Errorf("%s is not cacheable", host)
		}
	}
}

func TestAssetURLTakesThunderstoreIcons(t *testing.T) {
	if AssetURL("https://ccdn.thunderstore.io/live/repository/icons/Evaisa-HookGenPatcher-0.0.5.png") == "" {
		t.Fatal("a Thunderstore package icon is not served")
	}
}

func TestEnsureIgnoresEmpty(t *testing.T) {
	c := New(t.TempDir(), http.DefaultClient)
	c.Ensure(t.Context(), "")
	if entries, err := os.ReadDir(c.dir); err == nil && len(entries) != 0 {
		t.Fatal(entries)
	}
}

func TestPruneToDropsOldestOverBudget(t *testing.T) {
	c := New(t.TempDir(), http.DefaultClient)
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(c.dir, "old")
	newPath := filepath.Join(c.dir, "new")
	if err := os.WriteFile(oldPath, []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("67890"), 0o600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(oldPath, past, past); err != nil {
		t.Fatal(err)
	}
	c.pruneTo(6)
	if _, err := os.Stat(oldPath); err == nil {
		t.Fatal("kept the older file over budget")
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatal("dropped the newer file")
	}
}

func TestRecordWritePrunesPastBudget(t *testing.T) {
	c := New(t.TempDir(), http.DefaultClient)
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		t.Fatal(err)
	}
	const chunk = 16 << 20
	base := time.Now().Add(-time.Hour)
	for i := range 6 {
		p := filepath.Join(c.dir, string(rune('a'+i)))
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(p, chunk); err != nil {
			t.Fatal(err)
		}
		at := base.Add(time.Duration(i) * time.Minute)
		_ = os.Chtimes(p, at, at)
		c.recordWrite(chunk)
		var total int64
		entries, _ := os.ReadDir(c.dir)
		for _, e := range entries {
			info, _ := e.Info()
			total += info.Size()
		}
		if total > MaxCacheBytes {
			t.Fatalf("after write %d the folder holds %d bytes, over %d", i, total, MaxCacheBytes)
		}
	}
	if _, err := os.Stat(filepath.Join(c.dir, "a")); err == nil {
		t.Fatal("kept the oldest file")
	}
	if _, err := os.Stat(filepath.Join(c.dir, "f")); err != nil {
		t.Fatal("dropped the newest file")
	}
}
