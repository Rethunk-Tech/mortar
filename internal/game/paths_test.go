package game

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestStardewCatalogPathsMatchTheFolderStardewUses(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	config := filepath.Join(t.TempDir(), "config")
	appData := filepath.Join(t.TempDir(), "Roaming")
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("APPDATA", appData)
	t.Setenv("FLATPAK_ID", "")
	g, ok := catalogGame("stardew")
	if !ok {
		t.Fatal("stardew is not in the catalog")
	}
	flatpak := filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".config")
	cases := []struct {
		name, role, want string
		in               runtime.Install
	}{
		{"linux saves", PathSaves, filepath.Join(config, "StardewValley", "Saves"), runtime.Install{Store: StoreSteam, Platform: "linux", GOOS: "linux"}},
		{"flatpak steam saves", PathSaves, filepath.Join(flatpak, "StardewValley", "Saves"), runtime.Install{Store: StoreFlatpakSteam, Platform: "linux", GOOS: "linux"}},
		{"windows saves", PathSaves, filepath.Join(appData, "StardewValley", "Saves"), runtime.Install{Store: StoreSteam, Platform: "windows", GOOS: "windows"}},
		{"linux startup preferences", PathStartupPreferences, filepath.Join(config, "StardewValley", "startup_preferences"), runtime.Install{Store: StoreGOG, Platform: "linux", GOOS: "linux"}},
	}
	for _, c := range cases {
		c.in.Home = home
		got, err := runtime.Resolve(c.in, g.Paths[c.role])
		if err != nil || got != c.want {
			t.Fatalf("%s = %q, %v; want %q", c.name, got, err, c.want)
		}
	}
	got, err := SavesDir(home, settings.Defaults(), "stardew", "")
	if err != nil || got != filepath.Join(config, "StardewValley", "Saves") {
		t.Fatalf("SavesDir = %q, %v", got, err)
	}
}

func TestSavesDirNeedsACatalogPath(t *testing.T) {
	if HasSaves("nope") || HasStartupSettings("lethal-company") {
		t.Fatal("a game without the catalog path reported one")
	}
	if !HasSaves("lethal-company") {
		t.Fatal("lethal company's saves template is not found")
	}
}
