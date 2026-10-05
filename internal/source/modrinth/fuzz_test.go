package modrinth

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

// FuzzDecoders holds that no answer body from Modrinth makes search, resolve, versions or categories panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"total_hits":1,"hits":[{"project_id":"a","title":"T"}]}`))
	f.Add([]byte(`[{"id":"v","version_number":"1","dependencies":[{"project_id":"p","dependency_type":"required"}],"files":[{"hashes":{"sha512":"ab"},"primary":true}]}]`))
	f.Add([]byte(`[{"name":"fabric"}]`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{HTTP: &http.Client{Transport: canned(body)}, URL: "http://modrinth.test"}
		_, _ = d.Search(t.Context(), source.Query{Key: "fabric"})
		_, _ = d.Resolve(t.Context(), "p", "", "", nil)
		_, _ = d.Versions(t.Context(), "p", "")
		_, _ = d.Categories(t.Context(), "")
	})
}
