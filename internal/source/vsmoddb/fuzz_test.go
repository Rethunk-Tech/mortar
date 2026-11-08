package vsmoddb

import (
	"net/http"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testhttp"
)

// FuzzDecoders holds that no answer body from the Mod DB makes search, resolve or the update read panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"mods":[{"modid":1,"assetid":2,"name":"A","tags":["x"]}]}`))
	f.Add([]byte(`{"mod":{"modid":1,"assetid":2,"releases":[{"releaseid":3,"modversion":"1.0.0","mainfile":"https://c/f.zip"}]}}`))
	f.Add([]byte(`{"mod":null}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{HTTP: &http.Client{Transport: testhttp.Canned(body)}, URL: "http://moddb.test"}
		_, _ = d.Search(t.Context(), source.Query{Page: source.FirstPage + 1})
		_, _ = d.Resolve(t.Context(), "1-2", "")
		_, _ = d.Items(t.Context(), "", "", []string{"1-2"})
	})
}
