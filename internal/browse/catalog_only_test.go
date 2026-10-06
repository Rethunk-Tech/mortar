package browse

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func gzJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(raw)
	_ = w.Close()
	return buf.Bytes()
}

// A game only a manifest lists (no bundled entry, no Go of its own) browses its Thunderstore community and installs a
// package into a profile, the path Browse's Add takes.
func TestACatalogOnlyGameBrowsesAndAdds(t *testing.T) {
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: "content-warning", Name: "Content Warning", Enabled: true, Marker: "Content Warning.exe",
		R2modmanFolder: "ContentWarning", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "profile", Root: "{profile}"}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "2881650"}},
		Loaders: []components.GameLoader{{ID: "bepinex5", Name: "BepInEx 5"}},
		Sources: []components.GameSource{{ID: "thunderstore", Key: "content-warning"}},
	})
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/c/content-warning/api/v1/package-listing-index/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(gzJSON(t, []string{srv.URL + "/chunk"}))
	})
	mux.HandleFunc("/chunk", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(gzJSON(t, []map[string]any{{
			"name": "MoreEmotes", "owner": "Ns", "package_url": srv.URL + "/c/content-warning/p/Ns/MoreEmotes/",
			"versions": []map[string]any{{"version_number": "1.0.0", "description": "emotes", "downloads": 5}},
		}}))
	})
	source.Register(thunderstore.Driver{URL: srv.URL, CacheDir: t.TempDir()})
	t.Cleanup(func() { source.Register(thunderstore.Driver{}) })

	s := &Service{Version: "1"}
	if got := s.SearchableSources("content-warning"); len(got) != 1 || got[0].ID != "thunderstore" {
		t.Fatalf("sources = %+v", got)
	}
	page, err := s.Search(t.Context(), "content-warning", "thunderstore", "emotes", 1, "", Filter{})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "Ns-MoreEmotes" {
		t.Fatalf("search = %+v, %v", page, err)
	}
	// Add resolves the package in the game's community (main.go's queue Closure reads this key).
	if key := share.SourceKeys("content-warning")["thunderstore"]; key != "content-warning" {
		t.Fatalf("thunderstore key = %q", key)
	}

	base := t.TempDir()
	profiles := profile.OpenIn(filepath.Join(base, "profiles"), store.OpenAt(filepath.Join(base, "store")))
	p, err := profiles.Create("content-warning", "CW")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
		"manifest.json": `{"name":"MoreEmotes","version_number":"1.0.0"}`, "MoreEmotes.dll": "x",
	})
	res, err := profiles.InstallSource("content-warning", p.ID, zip,
		profile.Source{Kind: profile.KindThunderstore, Name: "Ns-MoreEmotes", Version: "1.0.0"})
	if err != nil || len(res.Profile.Entries) != 1 || res.Profile.Entries[0].Mods[0].ID != "thunderstore:Ns-MoreEmotes" {
		t.Fatalf("install = %+v, %v", res, err)
	}
}
