package stardew

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/gamestore"
)

func TestSavesDirUsesStoreSpecificConfigRoots(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	config := filepath.Join(t.TempDir(), "config")
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("FLATPAK_ID", "")

	got, err := Game{}.SavesDir(gamestore.StoreSteam, home)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(config, "StardewValley", "Saves")
	if got != want {
		t.Fatalf("native Steam saves = %q, want %q", got, want)
	}

	got, err = Game{}.SavesDir(gamestore.StoreGOG, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("native GOG saves = %q, want %q", got, want)
	}

	got, err = Game{}.SavesDir(gamestore.StoreFlatpakSteam, home)
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".config", "StardewValley", "Saves")
	if got != want {
		t.Fatalf("Flatpak Steam saves = %q, want %q", got, want)
	}
}

func TestSavesDirUsesHostHomeWhenMortarRunsInFlatpak(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "sandbox-config"))
	t.Setenv("FLATPAK_ID", "tech.rethunk.Mortar")

	got, err := Game{}.SavesDir(gamestore.StoreSteam, home)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "StardewValley", "Saves")
	if got != want {
		t.Fatalf("Flatpak Mortar saves = %q, want %q", got, want)
	}
}
