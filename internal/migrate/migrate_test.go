package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectsStardropProfileAndReadsGameMods(t *testing.T) {
	home := filepath.Join("testdata", "stardrop", "home")
	mods := filepath.Join("testdata", "stardrop", "game", "Mods")

	sources, err := Detect(home, mods)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Kind != KindStardrop {
		t.Fatalf("sources = %#v", sources)
	}
	preview, err := Preview(home, mods, KindStardrop, "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Mods) != 1 {
		t.Fatalf("mods = %#v", preview.Mods)
	}
	mod := preview.Mods[0]
	if mod.UniqueID != "Example.Mod" || mod.Name != "Example Mod" || mod.Version != "1.2.3" || !mod.Enabled || mod.NexusModID != 123 {
		t.Fatalf("mod = %#v", mod)
	}
	if mod.SourcePath == "" {
		t.Fatal("Stardrop mod has no source path")
	}
}

func TestDetectsVortexProfileAndReadsStagingMods(t *testing.T) {
	home := filepath.Join("testdata", "vortex", "home")
	mods := filepath.Join(home, ".config", "Vortex", "game", "mods")

	sources, err := Detect(home, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Kind != KindVortex {
		t.Fatalf("sources = %#v", sources)
	}
	preview, err := Preview(home, mods, KindVortex, "profile-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Mods) != 1 {
		t.Fatalf("mods = %#v", preview.Mods)
	}
	mod := preview.Mods[0]
	if mod.UniqueID != "Example.Mod" || mod.Name != "Example Mod" || mod.Version != "1.2.3" || !mod.Enabled || mod.NexusModID != 123 {
		t.Fatalf("mod = %#v", mod)
	}
	if mod.SourcePath == "" {
		t.Fatal("Vortex mod has no source path")
	}
}

func TestStardropProfileCarriesItsOwnConfigCopy(t *testing.T) {
	home := filepath.Join("testdata", "stardrop", "home")
	mods := filepath.Join("testdata", "stardrop", "game", "Mods")

	preview, err := Preview(home, mods, KindStardrop, "Kept")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Mods) != 1 || !preview.Mods[0].Enabled || string(preview.Mods[0].Config) != `{ "Volume": 3 }` {
		t.Fatalf("mods = %#v", preview.Mods)
	}
}

func TestFolderModsSkipsModsSMAPIInstallsItself(t *testing.T) {
	dir := t.TempDir()
	for name, id := range map[string]string{"ConsoleCommands": "SMAPI.ConsoleCommands", "Real": "Someone.Real"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o750); err != nil {
			t.Fatal(err)
		}
		body := `{"Name":"` + name + `","UniqueID":"` + id + `","Version":"1.0.0"}`
		if err := os.WriteFile(filepath.Join(dir, name, "manifest.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mods, err := folderMods(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].UniqueID != "Someone.Real" {
		t.Fatalf("mods = %+v, want only Someone.Real", mods)
	}
}
