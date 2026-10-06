package github

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// FuzzDecoders holds that no answer body from GitHub makes search or details panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"total_count":1,"items":[{"full_name":"o/r","name":"r","owner":{"login":"o"},"stargazers_count":2,"topics":["stardew-valley-mod"]}]}`))
	f.Add([]byte(`{"full_name":"o/r","description":"d","license":{"spdx_id":"MIT"}}`))
	f.Add([]byte(`{"items":null}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, body []byte) {
		hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}}, nil
		})}
		d := &Driver{HTTP: hc, URL: "http://github.test"}
		_, _ = d.Search(t.Context(), source.Query{Key: "stardew-valley-mod", Text: "x"})
		_, _ = d.Details(t.Context(), "", "o/r", "1.0.0")
	})
}
