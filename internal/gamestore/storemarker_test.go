package gamestore

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-gamelocator/epic"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

func TestAStoreMarkerFindsABuildWhoseExecutableSitsElsewhereThanTheGamesMarker(t *testing.T) {
	root := t.TempDir()
	touchFile(t, filepath.Join(root, "epic", "_windowsnosteam", "win64", "X.exe"))
	touchFile(t, filepath.Join(root, "gog", "Garlic", "_windowsnosteam", "win64", "X.exe"))
	touchFile(t, filepath.Join(root, "xbox", "win", "x.exe"))
	launcher := t.TempDir()
	list := fmt.Sprintf(`{"InstallationList":[{"InstallLocation":%q,"AppName":"Garlic"}]}`, filepath.Join(root, "epic"))
	if err := os.WriteFile(filepath.Join(launcher, epic.ListFile), []byte(list), 0o600); err != nil {
		t.Fatal(err)
	}
	old := appxLocation
	t.Cleanup(func() { appxLocation = old })
	appxLocation = func(string) string { return filepath.Join(root, "xbox") }
	win := "_windowsnosteam/win64/X.exe"
	g := components.GameInfo{ID: "x", Name: "X", Marker: "X.exe", MarkerDir: "_windows/win64", Stores: components.GameStores{
		Epic: &components.EpicStore{AppNames: []string{"Garlic"}, Marker: win},
		GOG:  &components.GOGStore{ProductID: "1", Folder: "Garlic", Marker: win},
		Xbox: &components.XboxStore{Package: "Coffee.X", Marker: "win/x.exe"},
	}}
	roots := map[string][]string{LauncherEpic: {launcher}, LauncherGOG: {filepath.Join(root, "gog")}}
	got := Discover(t.TempDir(), roots, g)
	want := []Install{
		{Store: StoreGOG, Dir: filepath.Join(root, "gog", "Garlic")},
		{Store: StoreEpic, Dir: filepath.Join(root, "epic")},
		{Store: StoreXbox, Dir: filepath.Join(root, "xbox")},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if MarkerPath(g, StoreGOG) != win || MarkerPath(g, StoreEpicHeroic) != win || MarkerPath(g, StoreXbox) != "win/x.exe" || MarkerPath(g, StoreSteam) != "_windows/win64/X.exe" {
		t.Fatal("each store starts the executable its own marker names, and Steam the game's")
	}
}
