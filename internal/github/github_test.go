package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-httpkit/download"
	"github.com/Rethunk-Tech/go-safefs"
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
