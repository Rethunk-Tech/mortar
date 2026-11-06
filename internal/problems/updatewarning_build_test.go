package problems

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testcatalog"
)

// A game whose loader cannot read its version is compared by the Steam build of its install.
func TestUpdateWarningComparesTheSteamBuildWhenTheLoaderReadsNoVersion(t *testing.T) {
	const id = "build-game"
	testcatalog.AddGames(t, components.GameInfo{
		ID: id, Name: "Build Game", Enabled: true, Marker: "G.dll", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods", MaxDepth: map[string]int{"pkg": 1}}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "7"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
	})

	datadirtest.Use(t, t.TempDir())
	lib := t.TempDir()
	folder := filepath.Join(lib, "steamapps", "common", "Build Game")
	if err := os.MkdirAll(folder, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "G.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	build := func(n string) {
		manifest := "\"AppState\"\n{\n\t\"buildid\"\t\t\"" + n + "\"\n}\n"
		if err := os.WriteFile(filepath.Join(lib, "steamapps", "appmanifest_7.acf"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	build("100")
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders[id] = folder }); err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, id, "A")
	s := NewService(t.TempDir(), set, profiles, nil)

	if _, err := set.RecordLastPlayed(id, p.ID, time.Now(), ""); err != nil {
		t.Fatal(err)
	}
	if w, err := s.UpdateWarning(t.Context(), id, p.ID); err != nil || w.Changed {
		t.Fatalf("a launch with no build recorded yet warned: %+v, %v", w, err)
	}
	if _, err := set.RecordBuild(id, "100"); err != nil {
		t.Fatal(err)
	}
	if w, err := s.UpdateWarning(t.Context(), id, p.ID); err != nil || w.Changed {
		t.Fatalf("the same build warned: %+v, %v", w, err)
	}
	build("101")
	w, err := s.UpdateWarning(t.Context(), id, p.ID)
	if err != nil || !w.Changed || w.Recorded != "build 100" || w.Installed != "build 101" {
		t.Fatalf("a patch went unnoticed: %+v, %v", w, err)
	}
}
