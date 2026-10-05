package runtime

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

var stardewSaves = components.PathTemplate{
	Windows: "{appData}/StardewValley/Saves",
	Linux:   "{xdgConfig}/StardewValley/Saves",
	Darwin:  "{xdgConfig}/StardewValley/Saves",
}

func TestNativeLinuxStores(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	config := filepath.Join(t.TempDir(), "config")
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("FLATPAK_ID", "")
	inst := Install{Store: "steam", Platform: "linux", Home: home, GOOS: "linux"}

	cases := []struct {
		name    string
		store   string
		flatpak string
		want    string
	}{
		{"native steam", "steam", "", filepath.Join(config, "StardewValley", "Saves")},
		{"native gog", "gog", "", filepath.Join(config, "StardewValley", "Saves")},
		{"flatpak steam", "flatpak-steam", "", filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".config", "StardewValley", "Saves")},
		{"mortar in flatpak", "steam", "tech.rethunk.Mortar", filepath.Join(home, ".config", "StardewValley", "Saves")},
	}
	for _, c := range cases {
		t.Setenv("FLATPAK_ID", c.flatpak)
		inst.Store = c.store
		got, err := Resolve(inst, stardewSaves)
		if err != nil || got != c.want {
			t.Fatalf("%s = %q, %v; want %q", c.name, got, err, c.want)
		}
	}
}

func TestNativeWindowsUsesAppData(t *testing.T) {
	appData := filepath.Join(t.TempDir(), "Roaming")
	t.Setenv("APPDATA", appData)
	inst := Install{Store: "steam", Platform: "windows", Home: t.TempDir(), GOOS: "windows"}
	if IDOf(inst) != Native {
		t.Fatal("a Windows build on Windows must be native")
	}
	got, err := Resolve(inst, stardewSaves)
	if want := filepath.Join(appData, "StardewValley", "Saves"); err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

func TestProtonResolvesInsideThePrefix(t *testing.T) {
	library := filepath.Join(t.TempDir(), "SteamLibrary")
	dir := filepath.Join(library, "steamapps", "common", "Lethal Company")
	user := filepath.Join(library, "steamapps", "compatdata", "1966720", "pfx", "drive_c", "users", "steamuser")
	inst := Install{Store: "steam", Dir: dir, AppID: "1966720", Platform: PlatformOf("Lethal Company.exe", "linux"), Home: t.TempDir(), GOOS: "linux"}
	if IDOf(inst) != Proton {
		t.Fatalf("runtime = %s", IDOf(inst))
	}
	got, err := Resolve(inst, components.PathTemplate{Windows: "{localLow}/ZeekerssRBLX/Lethal Company"})
	want := filepath.Join(user, "AppData", "LocalLow", "ZeekerssRBLX", "Lethal Company")
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

func TestNativeRefusesForeignBuildAndMissingPath(t *testing.T) {
	inst := Install{Store: "gog", Platform: "windows", Home: t.TempDir(), GOOS: "linux"}
	if _, err := Resolve(inst, stardewSaves); err == nil {
		t.Fatal("a Windows build outside Steam resolved on Linux")
	}
	inst = Install{Platform: "linux", Home: t.TempDir(), GOOS: "linux"}
	if _, err := Resolve(inst, components.PathTemplate{Windows: "{appData}/x"}); err == nil {
		t.Fatal("a missing platform template resolved")
	}
}
