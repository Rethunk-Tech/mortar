package browse

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testcatalog"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testthunderstore"
)

// A game only a manifest lists (no bundled entry, no Go of its own) browses its Thunderstore community and installs a
// package into a profile, the path Browse's Add takes.
func TestACatalogOnlyGameBrowsesAndAdds(t *testing.T) {
	testcatalog.AddGames(t, components.GameInfo{
		ID: "catalog-only-game", Name: "Content Warning", Enabled: true, Marker: "Content Warning.exe",
		R2modmanFolder: "ContentWarning", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "profile", Root: "{profile}"}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "2881650"}},
		Loaders: []components.GameLoader{{ID: "bepinex5", Name: "BepInEx 5"}},
		Sources: []components.GameSource{{ID: "thunderstore", Key: "content-warning"}},
	})

	srv := testthunderstore.Serve(t, "content-warning", `[{
		"name": "MoreEmotes", "owner": "Ns", "package_url": "{base}/c/content-warning/p/Ns/MoreEmotes/",
		"versions": [{"version_number": "1.0.0", "description": "emotes", "downloads": 5}]
	}]`, nil, false)
	source.Register(thunderstore.Driver{URL: srv.URL, CacheDir: t.TempDir()})
	t.Cleanup(func() { source.Register(thunderstore.Driver{}) })

	s := &Service{Version: "1"}
	if got := s.SearchableSources("catalog-only-game"); len(got) != 1 || got[0].ID != "thunderstore" {
		t.Fatalf("sources = %+v", got)
	}
	page, err := s.Search(t.Context(), "catalog-only-game", "thunderstore", "emotes", 1, "", Filter{})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "Ns-MoreEmotes" {
		t.Fatalf("search = %+v, %v", page, err)
	}
	// Add resolves the package in the game's community (main.go's queue Closure reads this key).
	if key := share.SourceKeys("catalog-only-game")["thunderstore"]; key != "content-warning" {
		t.Fatalf("thunderstore key = %q", key)
	}

	base := t.TempDir()
	_, profiles := testenv.StoresIn(t, base)
	p, err := profiles.Create("catalog-only-game", "CW")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
		"manifest.json": `{"name":"MoreEmotes","version_number":"1.0.0"}`, "MoreEmotes.dll": "x",
	})
	res, err := profiles.InstallSource(t.Context(), "catalog-only-game", p.ID, zip,
		profile.Source{Kind: profile.KindThunderstore, Name: "Ns-MoreEmotes", Version: "1.0.0"})
	if err != nil || len(res.Profile.Entries) != 1 || res.Profile.Entries[0].Mods[0].ID != "thunderstore:Ns-MoreEmotes" {
		t.Fatalf("install = %+v, %v", res, err)
	}
}
