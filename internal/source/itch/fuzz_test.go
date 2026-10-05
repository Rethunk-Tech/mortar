package itch

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

type canned []byte

func (c canned) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(c)), Header: http.Header{}}, nil
}

// FuzzDecoders holds that no answer body from itch.io makes search, resolve or validate panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"games":[{"id":1,"title":"T","user":{"username":"u"}}]}`))
	f.Add([]byte(`{"uploads":[{"id":2,"filename":"a.zip","size":3}],"url":"https://x","user":{"username":"u"}}`))
	f.Add([]byte(`{"errors":["invalid key"]}`))
	f.Add([]byte(`[`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{HTTP: &http.Client{Transport: canned(body)}, URL: "http://itch.test", Key: func() (string, error) { return "k", nil }}
		_, _ = d.Search(t.Context(), source.Query{Key: "x"})
		_, _ = d.Resolve(t.Context(), "1", "")
		_, _ = d.Validate(t.Context(), "k")
	})
}
