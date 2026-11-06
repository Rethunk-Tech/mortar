// Package testhttp holds HTTP fakes for tests; it imports no mortar package, so any test package may use it.
package testhttp

import (
	"bytes"
	"io"
	"net/http"
)

// Canned is a RoundTripper that answers every request 200 with the same body.
type Canned []byte

// RoundTrip returns the canned body.
func (c Canned) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(c)), Header: http.Header{}}, nil
}
