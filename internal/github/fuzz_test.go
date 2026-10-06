package github

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// FuzzFetchReleases holds that no releases answer from GitHub makes reading it panic.
func FuzzFetchReleases(f *testing.F) {
	f.Add([]byte(`[{"tag_name":"v1.0.0","prerelease":false,"assets":[{"name":"a.zip","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/a.zip","size":3,"digest":"sha256:ab"}]}]`))
	f.Add([]byte(`{"message":"Not Found"}`))
	f.Add([]byte(`[{"assets":null}]`))
	f.Add([]byte(`[`))
	f.Fuzz(func(t *testing.T, body []byte) {
		hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}}, nil
		})}
		_, _ = FetchReleases(t.Context(), hc, "http://github.test/repos/o/r/releases")
	})
}
