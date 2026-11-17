package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/sharehost"
)

func TestMuxSitePackagesAPI(t *testing.T) {
	site := t.TempDir()
	if err := os.WriteFile(filepath.Join(site, "index.html"), []byte("home"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(site, "404.html"), []byte("missing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(site, "skyrim", "s"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(site, "skyrim", "s", "index.html"), []byte("share"), 0o600); err != nil {
		t.Fatal(err)
	}
	pkg := t.TempDir()
	if err := os.WriteFile(filepath.Join(pkg, "foo.txt"), []byte("pkg"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newMux(site, pkg, sharehost.New(sharehost.NewMemStore()))

	tests := []struct {
		name, path string
		wantStatus int
		wantBody   string
	}{
		{"site index", "/", http.StatusOK, "home"},
		{"site 404", "/no-such-page", http.StatusNotFound, "missing"},
		{"share code", "/skyrim/s/AbCdEfGh12", http.StatusOK, "share"},
		{"share version", "/skyrim/s/AbCdEfGh12/2", http.StatusOK, "share"},
		{"packages", "/packages/foo.txt", http.StatusOK, "pkg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status %d", rec.Code)
			}
			if rec.Body.String() != tt.wantBody {
				t.Fatalf("body %q", rec.Body.String())
			}
		})
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/shares", nil))
	if rec.Code == http.StatusNotFound {
		t.Fatalf("api not routed: %d", rec.Code)
	}
}
