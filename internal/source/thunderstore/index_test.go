package thunderstore

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestAnOversizeAnswerIsRefusedNotCutShort(t *testing.T) {
	t.Parallel()
	body := make([]byte, source.MaxBody+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	defer srv.Close()
	if _, err := (Driver{}).get(context.Background(), srv.URL, "ua"); !errors.Is(err, fsx.ErrTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestNoDowngrade(t *testing.T) {
	t.Parallel()
	req := func(raw string) *http.Request {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Request{URL: u}
	}
	via := func(raw string, n int) []*http.Request {
		out := make([]*http.Request, n)
		for i := range out {
			out[i] = req(raw)
		}
		return out
	}
	for _, tc := range []struct {
		name    string
		to      string
		via     []*http.Request
		refused bool
	}{
		{"https to http", "http://thunderstore.io/x", via("https://thunderstore.io/", 1), true},
		{"https to https", "https://cdn.thunderstore.io/x", via("https://thunderstore.io/", 1), false},
		{"http to http", "http://127.0.0.1/x", via("http://127.0.0.1/", 1), false},
		{"nine hops", "https://thunderstore.io/x", via("https://thunderstore.io/", 9), false},
		{"ten hops", "https://thunderstore.io/x", via("https://thunderstore.io/", maxRedirects), true},
	} {
		if err := noDowngrade(req(tc.to), tc.via); (err != nil) != tc.refused {
			t.Errorf("%s: err = %v, refused want %v", tc.name, err, tc.refused)
		}
	}
}
