package components

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// A listed version and older are broken, a newer one is fixed, and a game-version range limits the entry.
func TestAKnownBrokenEntryFlagsThroughItsVersion(t *testing.T) {
	l := BrokenList{Entries: []BrokenEntry{
		{IDs: BrokenIDs{Thunderstore: "Evaisa-LethalThings"}, Through: "0.10.13", Reason: "r"},
		{IDs: BrokenIDs{Nexus: 42}, Through: "2.0.0", GameVersions: ">=50", Reason: "r"},
	}}
	things := func(v string) BrokenMod {
		return BrokenMod{ID: mod.NewID(mod.FormatThunderstore, "evaisa-lethalthings"), Version: v}
	}
	for v, want := range map[string]bool{"0.10.13": true, "0.9.0": true, "0.10.14": false, "local build": false} {
		if _, got := l.Find(things(v), ""); got != want {
			t.Errorf("LethalThings %s: broken %v, want %v", v, got, want)
		}
	}
	page := BrokenMod{ID: mod.NewID(mod.FormatSMAPI, "Some.Mod"), NexusID: 42, Version: "1.0.0"}
	for game, want := range map[string]bool{"49": false, "50": true, "": true} {
		if _, got := l.Find(page, game); got != want {
			t.Errorf("Nexus 42 on game %q: broken %v, want %v", game, got, want)
		}
	}
}

// A list is used only when its signature verifies, and an older serial than the one cached is refused.
func TestLoadKnownBrokenVerifiesAndRefusesARollback(t *testing.T) {
	public, private := newKey(t)
	body := func(serial int) []byte {
		return fmt.Appendf(nil, `{"game":"lc","serial":%d,"entries":[{"ids":{"thunderstore":"A-B"},"through":"1.0.0","reason":"r"}]}`, serial)
	}
	served := body(5)
	sig := ed25519.Sign(private, served)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/lc.json.sig" {
			_, _ = w.Write(sig)
			return
		}
		_, _ = w.Write(served)
	}))
	t.Cleanup(server.Close)
	c := &Client{HTTP: server.Client(), KnownBrokenURL: server.URL + "/"}
	cache := &meta.Client{CacheDir: t.TempDir()}
	l, err := c.LoadKnownBroken(t.Context(), cache, public, "lc")
	if err != nil || l.Serial != 5 || len(l.Entries) != 1 {
		t.Fatalf("load = %+v, %v", l, err)
	}
	// A forged list is refused; the cached one stays in use.
	served = body(9)
	cache2 := &meta.Client{CacheDir: t.TempDir()}
	if _, err := c.LoadKnownBroken(t.Context(), cache2, public, "lc"); err == nil {
		t.Error("a list with a bad signature loaded")
	}
	// An older, validly signed list than the cached one is a rollback.
	served = body(3)
	sig = ed25519.Sign(private, served)
	cache.Now = func() time.Time { return time.Now().Add(48 * time.Hour) }
	if l, err := c.LoadKnownBroken(t.Context(), cache, public, "lc"); err != nil || l.Serial != 5 {
		t.Fatalf("after a rollback = %+v, %v (want the cached serial 5)", l, err)
	}
	// The floor outlives the cache entry: with the entry gone, serial 3 is still refused.
	if err := os.Remove(filepath.Join(cache.CacheDir, knownBroken.listName("lc"))); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LoadKnownBroken(t.Context(), cache, public, "lc"); !errors.Is(err, ErrRollback) {
		t.Fatalf("a rollback with the cache removed = %v", err)
	}
}

// Another list signed by the same key is not a catalog: it must not decode as an empty one.
func TestDecodeRefusesAKnownBrokenList(t *testing.T) {
	if _, err := Decode([]byte(`{"game":"lc","serial":9,"entries":[]}`)); err == nil {
		t.Fatal("a signed known-broken list decoded as a catalog")
	}
}

func TestADropInNeedsASupersededEntryWithAReplacement(t *testing.T) {
	entry := func(extra string) []byte {
		return []byte(`{"game":"lethal-company","serial":1,"entries":[{"ids":{"thunderstore":"Old-Loader"},"through":"9.9.9","reason":"r"` + extra + `}]}`)
	}
	if _, err := DecodeBrokenList(entry(`,"dropIn":true,"replacement":"Mortar-Loader"`), "lethal-company"); err == nil {
		t.Error("a dropIn on a broken entry decoded")
	}
	l, err := DecodeBrokenList(entry(`,"kind":"superseded","dropIn":true,"replacement":"Mortar-Loader"`), "lethal-company")
	if err != nil || l.DropIns()["old-loader"] != "Mortar-Loader" {
		t.Errorf("drop-ins %v, %v", l.DropIns(), err)
	}
}
