package modio

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

type canned []byte

func (c canned) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(c)), Header: http.Header{}}, nil
}

// FuzzDecoders holds that no answer body from mod.io makes search, details, resolve, update checks or validate panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(timberAPI))
	f.Add([]byte(`{"data":[` + timberAPI + `],"result_total":1}`))
	f.Add([]byte(`{"data":[{"id":5,"name":"Harmony"},{"tags":[]}]}`))
	f.Add([]byte(`{"data":[{"hidden":false,"tags":["Mod"]}]}`))
	f.Add([]byte(`[`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{HTTP: &http.Client{Transport: canned(body)}, URL: "http://modio.test", Token: func() (string, error) { return "t", nil }}
		_, _ = d.Search(t.Context(), source.Query{Key: "1"})
		_, _ = d.Categories(t.Context(), "1")
		_, _ = d.Details(t.Context(), "1", "2", "")
		_, _ = d.Resolve(t.Context(), "1", "2", "3")
		_, _ = d.Latest(t.Context(), components.GameSource{Key: "1"}, "", []source.InstalledFile{{ID: "2", Digest: "md5:x"}})
		_, _ = d.Validate(t.Context(), "t")
	})
}
