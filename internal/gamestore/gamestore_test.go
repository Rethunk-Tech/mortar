package gamestore

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestBottlesFindsGamesInEveryBottleLocation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Bottles is Linux only")
	}
	home := t.TempDir()
	mk := func(parts ...string) {
		if err := os.MkdirAll(filepath.Join(parts...), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	touch := func(p string) {
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	native := filepath.Join(home, ".local", "share", "bottles", "bottles", "Games")
	flat := filepath.Join(home, ".var", "app", "com.usebottles.bottles", "data", "bottles", "bottles", "Gog")
	empty := filepath.Join(native, "..", "Empty")
	steamGame := filepath.Join(native, "drive_c", "Program Files (x86)", "Steam", "steamapps", "common", "Stardew Valley")
	gogGame := filepath.Join(flat, "drive_c", "GOG Games", "Stardew Valley")
	for _, d := range []string{steamGame, gogGame, empty} {
		mk(d)
	}
	touch(filepath.Join(steamGame, "Stardew Valley.dll"))
	touch(filepath.Join(gogGame, "Stardew Valley.dll"))
	touch(filepath.Join(native, "bottle.yml"))
	touch(filepath.Join(flat, "bottle.yml"))
	touch(filepath.Join(empty, "bottle.yml"))

	g := components.GameInfo{
		ID: "x", Name: "X", Marker: "Stardew Valley.dll",
		Stores: components.GameStores{Bottles: &components.BottlesStore{Folder: "Stardew Valley"}},
	}
	got := Discover(home, nil, g)
	want := []Install{
		{Store: StoreBottles, Dir: steamGame, Prefix: native},
		{Store: StoreBottles, Dir: gogGame, Prefix: flat},
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
