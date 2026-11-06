// Package testhttp holds HTTP fakes for tests; it imports no mortar package, so any test package may use it.
package testhttp

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// LegacyProfile serves body as the Thunderstore profile code key and 404 for everything else, and returns its URL.
func LegacyProfile(t testing.TB, key, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/experimental/legacyprofile/get/"+key+"/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// Canned is a RoundTripper that answers every request 200 with the same body.
type Canned []byte

// RoundTrip returns the canned body.
func (c Canned) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(c)), Header: http.Header{}}, nil
}
