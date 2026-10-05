package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	if got, err := Resolve(inst, components.PathTemplate{Windows: "{localLow}/../../../../../../outside"}); err == nil {
		t.Fatalf("a template climbed out of the prefix to %q", got)
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

func TestVersionReadsConfigInfoThenCompatToolMapping(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "steamapps", "common", "Lethal Company")
	inst := Install{Store: "steam", Dir: dir, AppID: "1966720", Platform: "windows", GOOS: "linux"}
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := Version(inst); got != "" {
		t.Fatalf("no Steam data: %q", got)
	}
	write("config/config.vdf", "\"Software\"\n{\n\t\"CompatToolMapping\"\n\t{\n\t\t\"0\"\n\t\t{\n\t\t\t\"name\"\t\t\"proton_9\"\n\t\t}\n\t\t\"1966720\"\n\t\t{\n\t\t\t\"name\"\t\t\"GE-Proton9-5\"\n\t\t}\n\t}\n}\n")
	if got := Version(inst); got != "GE-Proton9-5" {
		t.Fatalf("mapping: %q", got)
	}
	inst.AppID = "42"
	if got := Version(inst); got != "proton_9" {
		t.Fatalf("default mapping: %q", got)
	}
	inst.AppID = "1966720"
	write("steamapps/compatdata/1966720/config_info", "9.0-4\nother\n")
	if got := Version(inst); got != "9.0-4" {
		t.Fatalf("config_info: %q", got)
	}
	inst.Platform = "linux"
	if got := Version(inst); got != "" {
		t.Fatalf("native install: %q", got)
	}
}
