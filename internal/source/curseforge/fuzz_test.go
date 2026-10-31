package curseforge

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

// FuzzDecoders holds that no answer body from CurseForge makes search, resolve, details or an update check panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"data":[{"id":1,"name":"T","latestFiles":[{"id":2,"isAvailable":true,"hashes":[{"value":"ab","algo":1}],"dependencies":[{"modId":3,"relationType":3}]}]}],"pagination":{"totalCount":1}}`))
	f.Add([]byte(`{"data":{"id":1,"allowModDistribution":false}}`))
	f.Add([]byte(`{"data":null}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{
			HTTP: &http.Client{Transport: canned(body)}, URL: "http://curseforge.test", Key: func() string { return "k" },
			GameSource: func(string) (components.GameSource, bool) { return components.GameSource{GameID: 1}, true },
		}
		_, _ = d.Search(t.Context(), source.Query{Game: "g", Key: "1"})
		_, _ = d.Resolve(t.Context(), "1", "")
		_, _ = d.Details(t.Context(), "", "1", "")
		_, _ = d.Latest(t.Context(), components.GameSource{}, "", []source.InstalledFile{{ID: "1", Digest: "sha1:ab"}})
	})
}
