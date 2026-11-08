package updatesvc

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLatestPrereleaseTagRefusesAnOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"tag_name":"`))
		_, _ = w.Write(bytes.Repeat([]byte("1"), 8<<20))
		_, _ = w.Write([]byte(`","prerelease":true}]`))
	}))
	defer srv.Close()
	_, err := prereleaseProviderFor(srv).latestPrereleaseTag(t.Context(), "0.1.0")
	if err == nil {
		t.Fatal("oversized GitHub releases body was accepted")
	}
}

func TestLatestPrereleaseTagPicksFirstNewerPrerelease(t *testing.T) {
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		_, _ = w.Write([]byte(`[
			{"tag_name":"v1.2.0-beta.1","prerelease":true,"draft":true},
			{"tag_name":"v1.1.0","prerelease":false,"draft":false},
			{"tag_name":"v1.1.0-rc.1","prerelease":true,"draft":false},
			{"tag_name":"v1.0.1-beta.1","prerelease":true,"draft":false}
		]`))
	}))
	defer srv.Close()
	tag, err := prereleaseProviderFor(srv).latestPrereleaseTag(t.Context(), "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.1.0-rc.1" {
		t.Fatalf("tag = %q, want v1.1.0-rc.1", tag)
	}
	if accept != "application/vnd.github+json" {
		t.Fatalf("Accept = %q", accept)
	}
}

func TestLatestPrereleaseTagInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	_, err := prereleaseProviderFor(srv).latestPrereleaseTag(t.Context(), "0.1.0")
	if err == nil {
		t.Fatal("invalid GitHub releases body was accepted")
	}
	if !strings.Contains(err.Error(), "github releases:") {
		t.Fatalf("error %q: want github releases prefix", err)
	}
}

func prereleaseProviderFor(srv *httptest.Server) *prereleaseProvider {
	return &prereleaseProvider{client: &http.Client{Transport: rewriteHost{base: srv.URL, next: srv.Client().Transport}}}
}

type rewriteHost struct {
	base string
	next http.RoundTripper
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := req.URL.Parse(r.base + req.URL.RequestURI())
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.URL = u
	clone.Host = u.Host
	return r.next.RoundTrip(clone)
}
