package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestMO2DetectTwoProfiles(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	inst := filepath.Join(home, "AppData", "Local", "ModOrganizer", "Stardew")
	if err := copyMO2Fixture(filepath.Join("testdata", "mo2"), inst); err != nil {
		t.Fatal(err)
	}
	sources, err := Detect(home, "", "stardew")
	if err != nil {
		t.Fatal(err)
	}
	var source SourceInfo
	for _, item := range sources {
		if item.Kind == KindMO2 {
			source = item
			break
		}
	}
	if source.Kind != KindMO2 || source.Name != "Mod Organizer 2 (Stardew)" {
		t.Fatalf("expected the MO2 instance Stardew, got %#v", sources)
	}
	if len(source.Profiles) != 2 {
		t.Fatalf("profiles: got %d want 2 (%#v)", len(source.Profiles), source.Profiles)
	}

	def, err := Preview(home, "", "stardew", KindMO2, "Default")
	if err != nil {
		t.Fatal(err)
	}
	coop, err := Preview(home, "", "stardew", KindMO2, "Coop")
	if err != nil {
		t.Fatal(err)
	}
	assertMO2Plan(t, def, "Default", map[string]bool{
		"Pathoschild.ContentPatcher": true,
		"Fixture.SomeLocal":          false,
	}, 1915)
	assertMO2Plan(t, coop, "Coop", map[string]bool{
		"Pathoschild.ContentPatcher": false,
		"Fixture.SomeLocal":          true,
	}, 1915)
}

func assertMO2Plan(t *testing.T, preview ProfilePreview, name string, enabled map[string]bool, nexusID int) {
	t.Helper()
	if preview.Name != name || preview.Source != KindMO2 {
		t.Fatalf("preview identity: %+v", preview)
	}
	got := map[string]ModPreview{}
	for _, mp := range preview.Mods {
		got[mp.ID.Local()] = mp
	}
	if len(got) != len(enabled) {
		t.Fatalf("%s mods: got %#v want %d entries", name, preview.Mods, len(enabled))
	}
	for id, wantEnabled := range enabled {
		mod, ok := got[id]
		if !ok {
			t.Fatalf("%s missing %s in %#v", name, id, preview.Mods)
		}
		if mod.Enabled != wantEnabled {
			t.Fatalf("%s %s enabled=%v want %v", name, id, mod.Enabled, wantEnabled)
		}
		if id == "Pathoschild.ContentPatcher" && mod.NexusModID != nexusID {
			t.Fatalf("nexus id=%d want %d", mod.NexusModID, nexusID)
		}
		if id == "Fixture.SomeLocal" {
			if mod.NexusModID != 0 {
				t.Fatalf("local mod nexus id=%d", mod.NexusModID)
			}
			if mod.SourcePath == "" {
				t.Fatal("local mod missing SourcePath")
			}
		}
	}
}

func copyMO2Fixture(src, dest string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		if mkErr := os.MkdirAll(filepath.Dir(target), 0o750); mkErr != nil {
			return mkErr
		}
		data, readErr := fsx.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, data, 0o600)
	})
}

func TestMO2PortableIsNamedForTheProductNotTheFolder(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := copyMO2Fixture(filepath.Join("testdata", "mo2"), home); err != nil {
		t.Fatal(err)
	}
	sources, err := Detect(home, "", "stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Name != "Mod Organizer 2" {
		t.Fatalf("sources: %#v", sources)
	}
}
