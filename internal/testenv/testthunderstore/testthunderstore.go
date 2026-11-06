// Package testthunderstore serves a fake Thunderstore community; it imports no mortar package, so any test package may use it.
package testthunderstore

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Fake is a Thunderstore server whose package-listing index names one chunk.
type Fake struct {
	*httptest.Server
	// Downloads counts requests that were answered with a file.
	Downloads atomic.Int32
}

// Serve starts a server for community. Its listing index points at one chunk holding the JSON array chunk, in which
// "{base}" stands for the server's own URL, and each path in files is answered with its bytes. tls serves HTTPS; the
// caller's client is Fake.Client(). The server closes when the test ends.
func Serve(t testing.TB, community, chunk string, files map[string][]byte, tls bool) *Fake {
	t.Helper()
	f := &Fake{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/c/" + community + "/api/v1/package-listing-index/":
			_, _ = fmt.Fprintf(w, `[%q]`, f.URL+"/chunk")
		case "/chunk":
			_, _ = w.Write([]byte(strings.ReplaceAll(chunk, "{base}", f.URL)))
		default:
			body, ok := files[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			f.Downloads.Add(1)
			_, _ = w.Write(body)
		}
	})
	if tls {
		f.Server = httptest.NewTLSServer(handler)
	} else {
		f.Server = httptest.NewServer(handler)
	}
	t.Cleanup(f.Close)
	return f
}
