package migrate

import (
	"os"
	"path/filepath"
	"strings"
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
	sources, err := Detect(home, "", "stardew", "")
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

	def, err := Preview(home, "", "stardew", "", KindMO2, "Default")
	if err != nil {
		t.Fatal(err)
	}
	coop, err := Preview(home, "", "stardew", "", KindMO2, "Coop")
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
	sources, err := Detect(home, "", "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Name != "Mod Organizer 2" {
		t.Fatalf("sources: %#v", sources)
	}
}

func TestMO2DirsFollowSettings(t *testing.T) {
	t.Parallel()
	write := func(root, ini string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "ModOrganizer.ini"), []byte(ini), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	seed := func(base string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(base, "profiles", "Default"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "profiles", "Default", "modlist.txt"), []byte("+Some Mod\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("moved base", func(t *testing.T) {
		t.Parallel()
		root, base := t.TempDir(), filepath.Join(t.TempDir(), "MO2 data")
		write(root, "[General]\ngameName=Stardew Valley\n\n[Settings]\nbase_directory=@ByteArray("+filepath.ToSlash(base)+")\n")
		seed(base)
		profiles, mods, err := mo2Profiles(root, "", "Stardew Valley", "stardew")
		if err != nil || len(profiles) != 1 || mods != filepath.Join(base, "mods") {
			t.Fatalf("profiles=%d mods=%q err=%v", len(profiles), mods, err)
		}
	})

	t.Run("moved mod_directory", func(t *testing.T) {
		t.Parallel()
		root, mods, game := t.TempDir(), filepath.Join(t.TempDir(), "staged"), t.TempDir()
		if err := os.MkdirAll(mods, 0o750); err != nil {
			t.Fatal(err)
		}
		write(root, "[General]\ngameName=Stardew Valley\n\n[Settings]\nmod_directory=\""+strings.ReplaceAll(filepath.ToSlash(mods), "/", `\\`)+"\"\n")
		seed(root)
		_, got, err := mo2Profiles(root, game, "Stardew Valley", "stardew")
		if err != nil || got != mods {
			t.Fatalf("mods=%q want %q err=%v", got, mods, err)
		}
	})

	t.Run("BASE_DIR expands", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		write(root, "[Settings]\nprofiles_directory=%BASE_DIR%/prof\n")
		if _, profiles := mo2Dirs(root); profiles != filepath.Join(root, "prof") {
			t.Fatalf("profiles = %q", profiles)
		}
	})
}
