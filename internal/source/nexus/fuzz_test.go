package nexus

import (
	"net/http"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testhttp"
)

// FuzzDecoders holds that no GraphQL answer from Nexus makes search panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"data":{"mods":{"totalCount":1,"nodes":[{"modId":1,"name":"A","summary":"s","author":"u","endorsements":2,"downloads":3,"pictureUrl":"https://x/p.png","updatedAt":"2026-01-01T00:00:00Z","modCategory":{"name":"C"}}]}}}`))
	f.Add([]byte(`{"errors":[{"message":"bad"}]}`))
	f.Add([]byte(`{"data":null}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, body []byte) {
		d := Driver{HTTP: &http.Client{Transport: testhttp.Canned(body)}, URL: "http://nexus.test"}
		_, _ = d.Search(t.Context(), source.Query{Game: "stardew", Key: "stardewvalley", Text: "x"})
	})
}
