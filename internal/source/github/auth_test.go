package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	ghauth "github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Browse search and details carry the gh token too, or anonymous search's ten requests a minute fail the GitHub chip.
func TestRequestsCarryGhToken(t *testing.T) {
	old := ghauth.DefaultAuth
	ghauth.DefaultAuth = &ghauth.Auth{Run: func(context.Context) ([]byte, error) { return []byte("tok\n"), nil }}
	t.Cleanup(func() { ghauth.DefaultAuth = old })
	var paths []string
	d := &Driver{HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("%s: Authorization %q", r.URL.Path, got)
		}
		paths = append(paths, r.URL.Path)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"items":[]}`)), Request: r}, nil
	})}}
	if _, err := d.Search(context.Background(), source.Query{Key: "k", Text: "x", Page: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Details(context.Background(), "", "o/r", ""); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("requests %v", paths)
	}
}
