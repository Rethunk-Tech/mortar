package paradox

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

type canned []byte

func (c canned) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(c)), Header: http.Header{}}, nil
}

// FuzzDecoders holds that no answer body from Paradox Mods makes search, resolve, the update read, details or the
// changelog panic, and that a download address is only ever built from a name of safe characters.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"result":"OK","totalCount":1,"mods":[{"modId":"1","displayName":"A","tags":["x"]}]}`))
	f.Add([]byte(`{"result":"OK","modDetail":{"game":"cities_skylines_2","name":"a-b","modId":"1","requirements":[{"productType":"MOD","id":"2"}],"changelog":[{"modVersion":1,"details":"<p>x"}]}}`))
	f.Add([]byte(`{"result":"OK","modVersions":[{"id":1,"size":2,"userVersion":"1.0"}]}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{HTTP: &http.Client{Transport: canned(body)}, URL: "http://paradox.test"}
		_, _ = d.Search(t.Context(), source.Query{Key: "cities_skylines_2"})
		_, _ = d.Items(t.Context(), "", "", []string{"1"})
		_, _ = d.Details(t.Context(), "", "1", "")
		_, _ = d.Changelog(t.Context(), "", "1", "1", "2", "")
		if r, err := d.Resolve(t.Context(), "1", ""); err == nil {
			u, perr := url.Parse(r.URL)
			if perr != nil || u.Host != "modscontent.paradox-interactive.com" || strings.Contains(u.Path, "..") || u.RawQuery != "" {
				t.Fatalf("resolved %+v", r)
			}
		}
	})
}
