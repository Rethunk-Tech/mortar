package components

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

func TestV2SignatureDoesNotVerifyAsAnotherList(t *testing.T) {
	public, private := newKey(t)
	body, _ := signedManifest(t, 1, private)
	sig := SignV2(private, ListCatalog, body)
	if err := VerifyV2(body, sig, public, ListCatalog); err != nil {
		t.Fatalf("catalog v2: %v", err)
	}
	if err := VerifyV2(body, sig, public, ListSuppressions); err == nil {
		t.Fatal("a catalog v2 signature verified as suppressions")
	}
	if err := VerifyV2(body, sig, public, ListKnownBroken); err == nil {
		t.Fatal("a catalog v2 signature verified as known-broken")
	}
}

func TestAListWithOnlyALegacySignatureStillLoads(t *testing.T) {
	public, private := newKey(t)
	body, signature := signedManifest(t, bundledSerial(t)+1, private)
	if _, err := signedServer(t, body, signature).Load(t.Context(), nil, public); err != nil {
		t.Fatalf("catalog with only .sig: %v", err)
	}
	list := fmt.Appendf(nil, `{"game":"lc","serial":1,"entries":[{"ids":{"thunderstore":"A-B"},"through":"1.0.0","reason":"r"}]}`)
	sig := ed25519.Sign(private, list)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lc.json.sig":
			_, _ = w.Write(sig)
		case "/lc.json":
			_, _ = w.Write(list)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	c := &Client{HTTP: server.Client(), KnownBrokenURL: server.URL + "/"}
	got, err := c.LoadKnownBroken(t.Context(), &meta.Client{CacheDir: t.TempDir()}, public, "lc")
	if err != nil || got.Serial != 1 {
		t.Fatalf("known-broken with only .sig = %+v, %v", got, err)
	}
}

func TestAWrongV2SignatureIsRefusedWhenLegacyIsValid(t *testing.T) {
	public, private := newKey(t)
	body, signature := signedManifest(t, bundledSerial(t)+1, private)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/components.json.v2.sig":
			_, _ = w.Write([]byte("not-a-v2-signature"))
		case "/components.json.sig":
			_, _ = w.Write(signature)
		case "/components.json":
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.ManifestURL = server.URL + "/components.json"
	if _, err := client.Load(t.Context(), nil, public); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("catalog with a bad v2 signature: %v", err)
	}

	list := fmt.Appendf(nil, `{"game":"lc","serial":1,"entries":[{"ids":{"thunderstore":"A-B"},"through":"1.0.0","reason":"r"}]}`)
	legacy := ed25519.Sign(private, list)
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lc.json.v2.sig":
			_, _ = w.Write(SignV2(private, ListCatalog, list))
		case "/lc.json.sig":
			_, _ = w.Write(legacy)
		case "/lc.json":
			_, _ = w.Write(list)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(broken.Close)
	c := &Client{HTTP: broken.Client(), KnownBrokenURL: broken.URL + "/"}
	if _, err := c.LoadKnownBroken(t.Context(), &meta.Client{CacheDir: t.TempDir()}, public, "lc"); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("known-broken with a catalog v2 signature: %v", err)
	}
}

func TestAnOldCacheEntryLoads(t *testing.T) {
	public, private := newKey(t)
	body, signature := signedManifest(t, bundledSerial(t)+1, private)
	cache := &meta.Client{CacheDir: t.TempDir()}
	if _, err := diskcache.CachedStamped(cache.Cache(), cacheName, appversion.Build(), day, func() (cachedManifest, error) {
		return cachedManifest{Manifest: body, Signature: signature}, nil
	}); err != nil {
		t.Fatal(err)
	}
	client := NewClient(nil)
	if !client.UseCached(cache, public) || client.Manifest().Serial != bundledSerial(t)+1 {
		t.Fatal("an old cache entry was not selected")
	}

	list := fmt.Appendf(nil, `{"game":"lc","serial":4,"entries":[{"ids":{"thunderstore":"A-B"},"through":"1.0.0","reason":"r"}]}`)
	sig := ed25519.Sign(private, list)
	lists := &meta.Client{CacheDir: t.TempDir()}
	if _, err := diskcache.CachedStamped(lists.Cache(), knownBroken.listName("lc"), appversion.Build(), signedListTTL, func() (cachedSignedList, error) {
		return cachedSignedList{List: list, Signature: sig}, nil
	}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)
	c := &Client{HTTP: server.Client(), KnownBrokenURL: server.URL + "/"}
	got, err := c.LoadKnownBroken(t.Context(), lists, public, "lc")
	if err != nil || got.Serial != 4 {
		t.Fatalf("old known-broken cache = %+v, %v", got, err)
	}
}

func TestLoadPrefersAV2SignatureOnRefresh(t *testing.T) {
	public, private := newKey(t)
	body, legacy := signedManifest(t, bundledSerial(t)+1, private)
	v2 := SignV2(private, ListCatalog, body)
	var serveV2 bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/components.json.v2.sig":
			if !serveV2 {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(v2)
		case "/components.json.sig":
			_, _ = w.Write(legacy)
		case "/components.json":
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.ManifestURL = server.URL + "/components.json"
	cache := &meta.Client{CacheDir: t.TempDir()}
	if _, err := client.Load(t.Context(), cache, public); err != nil {
		t.Fatal(err)
	}
	got, ok := diskcache.Peek[cachedManifest](cache.Cache(), cacheName)
	if !ok || got.Value.Form != sigFormLegacy {
		t.Fatalf("form after legacy fetch = %q", got.Value.Form)
	}
	serveV2 = true
	cache.Now = func() time.Time { return time.Now().Add(48 * time.Hour) }
	if _, err := client.Load(t.Context(), cache, public); err != nil {
		t.Fatal(err)
	}
	got, ok = diskcache.Peek[cachedManifest](cache.Cache(), cacheName)
	if !ok || got.Value.Form != sigFormV2 {
		t.Fatalf("form after v2 refresh = %q", got.Value.Form)
	}
}
