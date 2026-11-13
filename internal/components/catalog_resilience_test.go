package components

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// releasesServer lists components releases newest first, each served from /<name>.json with its signature.
func releasesServer(t *testing.T, files map[string][2][]byte, order ...string) *Client {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases" {
			var items []string
			for i, name := range order {
				items = append(items, fmt.Sprintf(`{"tag_name":"components-v2-%d","assets":[{"name":"components.json","browser_download_url":"%s/%s/components.json"}]}`, 100-i, server.URL, name))
			}
			_, _ = fmt.Fprintf(w, "[%s]", strings.Join(items, ","))
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		f, ok := files[parts[0]]
		if !ok || len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		if parts[1] == "components.json.sig" {
			_, _ = w.Write(f[1])
			return
		}
		if parts[1] != "components.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(f[0])
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.ReleasesURL = server.URL + "/releases"
	return client
}

func TestManifestURLsReadsTheGitHubReleaseList(t *testing.T) {
	var accept string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		_, _ = fmt.Fprintf(w, `[{"tag_name":"v1","assets":[{"name":"components.json","browser_download_url":"%s/skip"}]},{"tag_name":"components-v2-1","draft":true,"assets":[{"name":"components.json","browser_download_url":"%s/draft"}]},{"tag_name":"components-v2-2","assets":[{"name":"other.json","browser_download_url":"%s/other"},{"name":"components.json","browser_download_url":"%s/good"}]}]`, server.URL, server.URL, server.URL, server.URL)
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.ReleasesURL = server.URL
	urls, err := client.manifestURLs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if accept != "application/vnd.github+json" {
		t.Fatalf("Accept = %q", accept)
	}
	if len(urls) != 1 || urls[0] != server.URL+"/good" {
		t.Fatalf("urls = %v", urls)
	}
}

func TestManifestURLsKeepsComponentManifestResponseWording(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "x", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.ReleasesURL = server.URL
	_, err := client.manifestURLs(t.Context())
	if err == nil || err.Error() != "component manifest response: server answered 502 Bad Gateway" {
		t.Fatalf("got %v", err)
	}
	var se *httpkit.StatusError
	if !errors.As(err, &se) || se.Code != http.StatusBadGateway {
		t.Fatalf("type %v", err)
	}
}

func TestLoadSkipsANewerReleaseThatDoesNotVerify(t *testing.T) {
	public, private := newKey(t)
	base := bundledSerial(t)
	good, goodSig := signedManifest(t, base+3, private)
	bad, badSig := signedManifest(t, base+9, private)
	badSig[0] ^= 0xff
	client := releasesServer(t, map[string][2][]byte{"good": {good, goodSig}, "bad": {bad, badSig}}, "bad", "good")
	cache := &meta.Client{CacheDir: t.TempDir()}
	manifest, err := client.Load(t.Context(), cache, public)
	if err != nil || manifest.Serial != base+3 {
		t.Fatalf("Load = serial %d, %v; the unverifiable newest release blocked the refresh", manifest.Serial, err)
	}
}

func TestTheSerialFloorOutlivesTheCachedCatalog(t *testing.T) {
	public, private := newKey(t)
	base := bundledSerial(t)
	newer, newerSig := signedManifest(t, base+5, private)
	older, olderSig := signedManifest(t, base+2, private)
	cache := &meta.Client{CacheDir: t.TempDir()}
	if _, err := releasesServer(t, map[string][2][]byte{"n": {newer, newerSig}}, "n").Load(t.Context(), cache, public); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(cache.CacheDir, cacheName)); err != nil {
		t.Fatal(err)
	}
	_, err := releasesServer(t, map[string][2][]byte{"o": {older, olderSig}}, "o").Load(t.Context(), cache, public)
	if err == nil || !strings.Contains(err.Error(), ErrRollback.Error()) {
		t.Fatalf("an older signed catalog was accepted after the cache entry went: %v", err)
	}
}

func TestComponentNamesThatLeaveTheReleaseFolderAreRefused(t *testing.T) {
	for _, tc := range []struct{ tag, asset string }{{"v1/../x", "a.zip"}, {"v1", "../a.zip"}, {"v1", "a?b.zip"}, {"..", "a.zip"}, {"v1", "a#b"}} {
		c := Component{
			Game: "g", Name: "n", Kind: "loader", Version: "1", Tag: tc.tag, Asset: tc.asset,
			Source: Source{Host: "github.com", Owner: "o", Repo: "r"},
			SHA256: strings.Repeat("a", 64),
		}
		if err := c.Validate(); err == nil {
			t.Errorf("tag %q asset %q accepted", tc.tag, tc.asset)
		}
	}
}

func TestPinsAreReadByTheCatalogAndAnInvalidOneIsDropped(t *testing.T) {
	good := Pin{Kind: PinPack, Name: "BepInEx-BepInExPack", Version: "5.4.2100", SHA256: strings.Repeat("ab", 32)}
	bad := Pin{Kind: "future-kind", Name: "x", Version: "1", SHA256: strings.Repeat("ab", 32)}
	m, dropped := withoutUnreadable(Manifest{Pins: []Pin{good, bad}})
	if dropped != 0 || len(m.Pins) != 1 || m.Pins[0] != good {
		t.Fatalf("pins = %+v", m.Pins)
	}
	c := NewClient(nil)
	c.SetManifest(m)
	if sum, ok := c.Pin(PinPack, "", "bepinex-bepinexpack", "5.4.2100"); !ok || sum != good.SHA256 {
		t.Fatalf("pin = %q, %v", sum, ok)
	}
	if _, ok := c.Pin(PinPack, "", "BepInEx-BepInExPack", "9"); ok {
		t.Fatal("an unpinned version was found")
	}
}
