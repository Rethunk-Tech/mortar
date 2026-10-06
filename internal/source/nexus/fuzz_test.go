package nexus

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

// FuzzDecoders holds that no GraphQL answer from Nexus makes search panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"data":{"mods":{"totalCount":1,"nodes":[{"modId":1,"name":"A","summary":"s","author":"u","endorsements":2,"downloads":3,"pictureUrl":"https://x/p.png","updatedAt":"2026-01-01T00:00:00Z","modCategory":{"name":"C"}}]}}}`))
	f.Add([]byte(`{"errors":[{"message":"bad"}]}`))
	f.Add([]byte(`{"data":null}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{HTTP: &http.Client{Transport: canned(body)}, URL: "http://nexus.test"}
		_, _ = d.Search(t.Context(), source.Query{Game: "stardew", Key: "stardewvalley", Text: "x"})
	})
}
