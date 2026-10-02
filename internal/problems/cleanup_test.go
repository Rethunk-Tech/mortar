package problems

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/manifest"
)

func TestCleanupHintsFindsUnusedFrameworkNeededOnlyByDisabledMod(t *testing.T) {
	spaceCore := manifest.Manifest{
		Name:     "SpaceCore",
		UniqueID: "spacechase0.SpaceCore",
	}
	oldPack := manifest.Manifest{
		UniqueID: "Example.OldPack",
		Dependencies: []manifest.Dependency{{
			UniqueID: "spacechase0.SpaceCore",
			Required: false,
		}},
	}
	contentFramework := manifest.Manifest{
		Name:     "Content Framework",
		UniqueID: "Example.ContentFramework",
	}
	oldContentPack := manifest.Manifest{
		UniqueID:       "Example.OldContentPack",
		ContentPackFor: "Example.ContentFramework",
	}
	mods := []Installed{
		{
			Key:      "spacecore",
			Enabled:  true,
			Manifest: spaceCore,
		},
		{
			Key:      "old-pack",
			Enabled:  false,
			Manifest: oldPack,
		},
		{
			Key:      "content-framework",
			Enabled:  true,
			Manifest: contentFramework,
		},
		{
			Key:      "old-content-pack",
			Enabled:  false,
			Manifest: oldContentPack,
		},
	}

	got := cleanupHints(mods)
	if len(got) != 2 || got[0].Key != "content-framework" || got[1].Key != "spacecore" {
		t.Fatalf("cleanupHints() = %#v, want both unused frameworks", got)
	}
}

func TestCleanupHintsTilesheets(t *testing.T) {
	t.Run("tmx use", func(t *testing.T) {
		pack := tilesheetPack(t, "Tiles.TMX", `{"Changes":[{"Action":"Load","Target":"Maps/SharedTiles"}]}`)
		other := modFolder(t, "Map User", "map.tmx", `<map><tileset><image source="Maps/SharedTiles.png"/></tileset></map>`)
		if got := cleanupHints([]Installed{pack, other}); len(got) != 0 {
			t.Fatalf("TMX use was flagged: %#v", got)
		}
	})
	t.Run("tbin use", func(t *testing.T) {
		pack := tilesheetPack(t, "Tiles.TBIN", `{"Changes":[{"Action":"Load","Target":"TileSheets/SharedTiles"}]}`)
		other := modFolder(t, "Binary User", "map.tbin", "prefix sharedtiles suffix")
		if got := cleanupHints([]Installed{pack, other}); len(got) != 0 {
			t.Fatalf("TBIN use was flagged: %#v", got)
		}
	})
	t.Run("optional dependency", func(t *testing.T) {
		pack := tilesheetPack(t, "Tiles.Optional", `{"Changes":[{"Action":"Load","Target":"Maps/OptionalTiles"}]}`)
		other := Installed{Key: "optional-user", Enabled: true, Manifest: manifest.Manifest{
			Name: "Optional User", UniqueID: "Other.Optional", Dependencies: []manifest.Dependency{{UniqueID: "Tiles.Optional", Required: false}},
		}}
		if got := cleanupHints([]Installed{pack, other}); len(got) != 0 {
			t.Fatalf("optional dependency was flagged: %#v", got)
		}
	})
	t.Run("unreferenced", func(t *testing.T) {
		pack := tilesheetPack(t, "Tiles.Unused", `{"Changes":[{"Action":"Load","Target":"Maps/UnusedTiles"}]}`)
		got := cleanupHints([]Installed{pack})
		if len(got) != 1 || got[0].Reason != "Tilesheets not used by any installed mod" {
			t.Fatalf("unreferenced tilesheet = %#v", got)
		}
	})
	t.Run("disabled use", func(t *testing.T) {
		pack := tilesheetPack(t, "Tiles.Disabled", `{"Changes":[{"Action":"Load","Target":"Maps/DisabledTiles"}]}`)
		other := Installed{Key: "disabled-user", Enabled: false, Manifest: manifest.Manifest{
			Name: "Disabled User", UniqueID: "Other.Disabled", Dependencies: []manifest.Dependency{{UniqueID: "Tiles.Disabled", Required: false}},
		}}
		got := cleanupHints([]Installed{pack, other})
		if len(got) != 1 || got[0].Reason != "Used only by switched-off mods: Disabled User" {
			t.Fatalf("disabled tilesheet = %#v", got)
		}
	})
	t.Run("edit data is not a tilesheet", func(t *testing.T) {
		pack := tilesheetPack(t, "Tiles.EditData", `{"Changes":[{"Action":"Load","Target":"Maps/EditDataTiles"},{"Action":"EditData","Target":"Data/Locations"}]}`)
		if got := cleanupHints([]Installed{pack}); len(got) != 0 {
			t.Fatalf("EditData pack was flagged: %#v", got)
		}
	})
}

func tilesheetPack(t *testing.T, id, content string) Installed {
	t.Helper()
	root := t.TempDir()
	writeProblemFile(t, root, "manifest.json", `{"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	writeProblemFile(t, root, "content.json", content)
	return Installed{Key: id, Enabled: true, Folder: root, Manifest: manifest.Manifest{Name: id, UniqueID: id}}
}

func modFolder(t *testing.T, name, file, content string) Installed {
	t.Helper()
	root := t.TempDir()
	writeProblemFile(t, root, file, content)
	return Installed{Key: name, Enabled: true, Folder: root, Manifest: manifest.Manifest{Name: name, UniqueID: name}}
}

func writeProblemFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
