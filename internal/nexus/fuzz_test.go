package nexus

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

type canned []byte

func (c canned) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(c)), Header: http.Header{}}, nil
}

// FuzzDecoders holds that no answer body from the Nexus API, v1 or GraphQL, makes a read panic. Each input runs
// every read, so minimising a new input takes most of the default minute: run it with -fuzzminimizetime=200x.
func FuzzDecoders(f *testing.F) {
	for _, name := range []string{"mod-541.json", "files-541.json", "changelogs-541.json", "game.json", "validate.json", "download-link.json"} {
		f.Add(fixture(f, name))
	}
	f.Add([]byte(`{"data":{"modFiles":[{"fileId":12,"scannedV2":"QUARANTINED"}]}}`))
	f.Add([]byte(`{"data":{"legacyModsByDomain":{"nodes":[{"modId":1,"name":"A","modRequirements":{"nexusRequirements":{"nodes":[{"modId":"2"}]}}}]}}}`))
	f.Add([]byte(`{"errors":[{"message":"x"}]}`))
	f.Add([]byte(`[`))
	f.Fuzz(func(t *testing.T, body []byte) {
		c := New("1.0")
		c.HTTP, c.BaseURL, c.CacheDir = &http.Client{Transport: canned(body)}, "http://nexus.test", t.TempDir()
		c = c.WithKey("k")
		ctx := t.Context()
		_, _ = c.Validate(ctx)
		_, _ = c.Mod(ctx, stardew, 1)
		_, _ = c.Page(ctx, stardew, 1)
		_, _ = c.Files(ctx, stardew, 1)
		_, _ = c.Changelogs(ctx, stardew, 1, 5)
		_, _ = c.Categories(ctx, stardew)
		_, _ = c.DownloadLinks(ctx, stardew, 1, 2, "", 0)
		_, _ = c.ScanStatuses(ctx, stardew, 1)
		_, _ = c.TrackedMods(ctx)
		_, _ = c.ModsByDomain(ctx, stardew.Domain, []int{1, 2})
		_, _ = c.FilesOf(ctx, stardew, []int{1})
	})
}
