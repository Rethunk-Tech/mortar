package host

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

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
	if b, _ := safefs.ReadFile(p); string(b) != "0123456789" {
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

func TestFileNameStaysOneSegment(t *testing.T) {
	u, err := url.Parse("https://example.com/files/..\\..\\evil.exe")
	if err != nil {
		t.Fatal(err)
	}
	got := fileName(u)
	dir := t.TempDir()
	if strings.ContainsAny(got, `/\`) || filepath.Dir(filepath.Join(dir, got)) != dir {
		t.Fatalf("hostile URL path became %q", got)
	}
}

func TestFetchRefusesARedirectToPlainHTTP(t *testing.T) {
	t.Parallel()
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("x")) }))
	t.Cleanup(plain.Close)
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL, http.StatusFound)
	}))
	t.Cleanup(secure.Close)
	_, err := (&Direct{HTTP: secure.Client()}).Fetch(t.Context(), secure.URL+"/m.zip", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "refusing a redirect") {
		t.Fatalf("err = %v", err)
	}
}
