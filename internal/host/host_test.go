package host

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestFor(t *testing.T) {
	cases := map[string]string{
		"https://github.com/o/r/releases/download/v1/a.zip": "github",
		"https://github.com/o/r":                            "direct",
		"https://example.com/a.zip":                         "direct",
		"http://example.com/a.zip":                          "",
		"ftp://x/a.zip":                                     "",
	}
	for u, want := range cases {
		got := ""
		if h := For(u, nil); h != nil {
			got = h.ID()
		}
		if got != want {
			t.Errorf("For(%q) = %q, want %q", u, got, want)
		}
	}
}

func TestDirectFetch(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/bad") {
			http.Error(w, "no", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer srv.Close()
	dir := t.TempDir()
	d := &Direct{HTTP: srv.Client(), MaxBytes: 10}

	p, err := d.Fetch(t.Context(), srv.URL+"/files/Mod.zip", dir)
	if err != nil || filepath.Base(p) != "Mod.zip" {
		t.Fatalf("fetch: %q %v", p, err)
	}
	if b, _ := fsx.ReadFile(p); string(b) != "0123456789" {
		t.Fatalf("body %q", b)
	}

	small := &Direct{HTTP: srv.Client(), MaxBytes: 9}
	if _, err := small.Fetch(t.Context(), srv.URL+"/files/Big.zip", dir); err == nil {
		t.Fatal("cap not enforced")
	}
	if _, err := d.Fetch(t.Context(), srv.URL+"/bad/Err.zip", dir); err == nil {
		t.Fatal("error status accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("failed fetches left files behind: %v", entries)
	}
}

func TestGitHubFetchGoesThroughGitHubDownload(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("zip")) }))
	defer srv.Close()
	var asked string
	g := &GitHub{HTTP: srv.Client(), Download: func(_ context.Context, _ *http.Client, url, dest string, _ int64, _ func(int64, int64)) error {
		asked = url
		return os.WriteFile(dest, []byte("zip"), 0o600)
	}}
	p, err := g.Fetch(t.Context(), srv.URL+"/o/r/releases/download/v1/a.zip", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(p); string(b) != "zip" || !strings.HasSuffix(asked, "/a.zip") {
		t.Fatalf("body %q", b)
	}
}

func TestDirectRefusesPlainHTTPRedirectsAndDotNames(t *testing.T) {
	var plainHit bool
	plain := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { plainHit = true }))
	defer plain.Close()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/redirect") {
			http.Redirect(w, r, plain.URL+"/secret", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("x"))
	}))
	defer srv.Close()
	dir := t.TempDir()
	d := &Direct{HTTP: srv.Client()}
	if _, err := d.Fetch(t.Context(), srv.URL+"/redirect", dir); err == nil || plainHit {
		t.Fatalf("followed a redirect to plain http: err %v, hit %v", err, plainHit)
	}
	p, err := d.Fetch(t.Context(), srv.URL+"/a/..", dir)
	if err != nil || filepath.Base(p) != "download" || filepath.Dir(p) != dir {
		t.Fatalf("a .. path must not name the file: %q %v", p, err)
	}
}
