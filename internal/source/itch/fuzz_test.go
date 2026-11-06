package itch

import (
	"net/http"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testhttp"
)

// FuzzDecoders holds that no answer body from itch.io makes search, resolve or validate panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"games":[{"id":1,"title":"T","user":{"username":"u"}}]}`))
	f.Add([]byte(`{"uploads":[{"id":2,"filename":"a.zip","size":3}],"url":"https://x","user":{"username":"u"}}`))
	f.Add([]byte(`{"errors":["invalid key"]}`))
	f.Add([]byte(`[`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{HTTP: &http.Client{Transport: testhttp.Canned(body)}, URL: "http://itch.test", Key: func() (string, error) { return "k", nil }}
		_, _ = d.Search(t.Context(), source.Query{Key: "x"})
		_, _ = d.Resolve(t.Context(), "1", "")
		_, _ = d.Validate(t.Context(), "k")
	})
}
