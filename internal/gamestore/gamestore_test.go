package gamestore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

func TestDriversRunOnlyForCatalogStores(t *testing.T) {
	home := t.TempDir()
	// A GOG offline-installer folder that would match if the GOG driver ran.
	dir := filepath.Join(home, "GOG Games", "Stardew Valley")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	roots := map[string][]string{LauncherGOG: {filepath.Join(home, "GOG Games")}}
	g := components.GameInfo{
		ID: "x", Name: "X", Marker: "Stardew Valley.dll",
		Stores: components.GameStores{GOG: &components.GOGStore{ProductID: "1", Folder: "Stardew Valley"}},
	}
	if got := Discover(home, roots, g); len(got) != 1 || got[0].Store != StoreGOG {
		t.Fatalf("a game naming gog must be found through it: %+v", got)
	}
	g.Stores.GOG = nil
	if got := Discover(home, roots, g); len(got) != 0 {
		t.Fatalf("a game without a gog key must never reach the GOG driver: %+v", got)
	}
}

func TestRankKeepsStoreOrder(t *testing.T) {
	if Rank(StoreSteam) >= Rank(StoreGOG) || Rank(StoreGOG) >= Rank(StoreLutris) || Rank("unknown") != Rank(StoreSteam) {
		t.Fatal("store order changed")
	}
}
