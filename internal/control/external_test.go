package control

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/migrate"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestExternalImportFindsAnotherManagersProfileByName(t *testing.T) {
	s := services(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.CopyFS(filepath.Join(home, "AppData", "Local", "ModOrganizer", "Stardew"), os.DirFS(filepath.Join("..", "migrate", "testdata", "mo2"))); err != nil {
		t.Fatal(err)
	}
	gameDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(gameDir, "Stardew Valley.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settings.Update(func(v *settings.Settings) { v.GameFolders["stardew"] = gameDir }); err != nil {
		t.Fatal(err)
	}
	got, err := s.Handle(t.Context(), "external.sources", Params{Game: "stardew", Source: migrate.KindMO2})
	if sources, ok := got.([]migrate.SourceInfo); err != nil || !ok || len(sources) != 1 || len(sources[0].Profiles) != 2 {
		t.Fatalf("sources = %#v, %v", got, err)
	}
	got, err = s.Handle(t.Context(), "external.sources", Params{Game: "stardew", Source: migrate.KindVortex})
	if sources, ok := got.([]migrate.SourceInfo); err != nil || !ok || len(sources) != 0 {
		t.Fatalf("vortex sources = %#v, %v", got, err)
	}
	got, err = s.Handle(t.Context(), "external.import", Params{Game: "stardew", Source: migrate.KindMO2, ID: "Coop", Preview: true})
	if pv, ok := got.(migrate.ProfilePreview); err != nil || !ok || pv.Name != "Coop" || len(pv.Mods) != 2 {
		t.Fatalf("preview = %#v, %v", got, err)
	}
	if _, err = s.Handle(t.Context(), "external.import", Params{Game: "stardew", Source: migrate.KindMO2, ID: "Nope", Preview: true}); usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("an unknown profile = %v", err)
	}
}

// A Vortex folder chosen after the app started is read by the very next listing, whatever install path
// the Vortex data names for the game.
func TestExternalSourcesListsVortexProfileFromChosenFolder(t *testing.T) {
	s := services(t)
	t.Setenv("HOME", t.TempDir())
	gameDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(gameDir, "Stardew Valley.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settings.Update(func(v *settings.Settings) { v.GameFolders["stardew"] = gameDir }); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	db, err := leveldb.OpenFile(filepath.Join(root, "state.v2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"persistent###profiles###p1###id":                         `"p1"`,
		"persistent###profiles###p1###gameId":                     `"stardewvalley"`,
		"persistent###profiles###p1###name":                       `"Default"`,
		"settings###gameMode###discovered###stardewvalley###path": `"C:\\Program Files (x86)\\Steam\\steamapps\\common\\Stardew Valley"`,
	} {
		if err := db.Put([]byte(k), []byte(v), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	p := Params{Game: "stardew", Source: migrate.KindVortex}
	got, _ := s.Handle(t.Context(), "external.sources", p)
	if sources, ok := got.([]migrate.SourceInfo); !ok || len(sources) != 0 {
		t.Fatalf("before choosing = %#v", got)
	}
	if _, err := s.Settings.Update(func(v *settings.Settings) { v.VortexFolder = root }); err != nil {
		t.Fatal(err)
	}
	got, err = s.Handle(t.Context(), "external.sources", p)
	if sources, ok := got.([]migrate.SourceInfo); err != nil || !ok || len(sources) != 1 || sources[0].Profiles[0].Name != "Default" {
		t.Fatalf("after choosing = %#v, %v", got, err)
	}
}
