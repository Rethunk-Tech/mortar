package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/github"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestBearerTransportAuthorizationHost(t *testing.T) {
	var got []struct{ host, auth string }
	rt := bearerTransport{
		token: "secret-token",
		next: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			got = append(got, struct{ host, auth string }{req.URL.Host, req.Header.Get("Authorization")})
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
		}),
	}
	for _, url := range []string{
		"https://api.github.com/repos/o/r/releases",
		"https://github.com/o/r/releases/download/v1/a.zip",
		"https://release-assets.githubusercontent.com/file",
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer leftover")
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	if len(got) != 3 {
		t.Fatalf("got %d requests", len(got))
	}
	if got[0].auth != "Bearer secret-token" || got[1].auth != "Bearer secret-token" {
		t.Fatalf("github hosts should carry the token: %+v", got[:2])
	}
	if got[2].auth != "" {
		t.Fatalf("cdn redirect host %q carried Authorization %q", got[2].host, got[2].auth)
	}
}

func TestSelectAsset(t *testing.T) {
	releases := []github.Release{
		{Tag: "v1.0.0", Draft: true, Assets: []github.Asset{{Name: "mod-1.0.0.zip"}}},
		{Tag: "v1.1.0-beta", Prerelease: true, Assets: []github.Asset{{Name: "mod-1.1.0-beta.zip"}}},
		{Tag: "v1.2.3", Assets: []github.Asset{{Name: "mod-1.2.3.zip"}, {Name: "mod-v1.2.3.zip"}}},
		{Tag: "v2.0.0", Assets: []github.Asset{{Name: "mod-2.0.0.zip"}}},
	}
	t.Run("skips drafts and prereleases for latest", func(t *testing.T) {
		rel, asset, err := selectAsset(releases, "o/r", "latest", "mod-{version}.zip")
		if err != nil {
			t.Fatal(err)
		}
		if rel.Tag != "v1.2.3" || asset.Name != "mod-1.2.3.zip" {
			t.Fatalf("got tag %q asset %q", rel.Tag, asset.Name)
		}
	})
	t.Run("honours v prefix", func(t *testing.T) {
		rel, _, err := selectAsset(releases, "o/r", "v1.2.3", "mod-{version}.zip")
		if err != nil {
			t.Fatal(err)
		}
		if rel.Tag != "v1.2.3" {
			t.Fatalf("got tag %q", rel.Tag)
		}
	})
	t.Run("expands version and tag", func(t *testing.T) {
		_, asset, err := selectAsset(releases, "o/r", "1.2.3", "mod-{tag}.zip")
		if err != nil {
			t.Fatal(err)
		}
		if asset.Name != "mod-v1.2.3.zip" {
			t.Fatalf("got asset %q", asset.Name)
		}
	})
	t.Run("invalid pattern", func(t *testing.T) {
		_, _, err := selectAsset(releases, "o/r", "latest", "[")
		if err == nil || !strings.Contains(err.Error(), "invalid asset pattern") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestResolveRejects(t *testing.T) {
	ctx := t.Context()
	client := &github.Client{}
	t.Run("missing field", func(t *testing.T) {
		_, err := resolve(ctx, client, components.SourceComponent{Name: "smapi", Kind: "loader"})
		if err == nil || !strings.Contains(err.Error(), "missing a required field") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unknown kind", func(t *testing.T) {
		src := components.SourceComponent{
			Game: "stardewvalley", Name: "smapi", Kind: "plugin", Version: "1.0.0", AssetPattern: "*.zip",
		}
		src.Source.Host, src.Source.Owner, src.Source.Repo = "github.com", "o", "r"
		_, err := resolve(ctx, client, src)
		if err == nil || !strings.Contains(err.Error(), "unknown kind") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("non-github host", func(t *testing.T) {
		src := components.SourceComponent{
			Game: "stardewvalley", Name: "smapi", Kind: "loader", Version: "1.0.0", AssetPattern: "*.zip",
		}
		src.Source.Host, src.Source.Owner, src.Source.Repo = "gitlab.com", "o", "r"
		_, err := resolve(ctx, client, src)
		if err == nil || !strings.Contains(err.Error(), "gitlab.com") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestNextSerial(t *testing.T) {
	dir := t.TempDir()
	t.Run("previous plus one when ahead of the clock", func(t *testing.T) {
		path := filepath.Join(dir, "ahead.json")
		const previous uint64 = 9_999_999_999
		if err := os.WriteFile(path, []byte(`{"serial":9999999999}`), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := nextSerial(path)
		if err != nil {
			t.Fatal(err)
		}
		if got != previous+1 {
			t.Fatalf("got %d want %d", got, previous+1)
		}
	})
	t.Run("corrupt previous file", func(t *testing.T) {
		path := filepath.Join(dir, "corrupt.json")
		if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := nextSerial(path)
		if err != nil {
			t.Fatal(err)
		}
		if now := time.Now().Unix(); got == 0 || int64(got) > now+1 { //nolint:gosec // G115: a serial is a Unix time, far below int64's range
			t.Fatalf("got serial %d", got)
		}
	})
}
