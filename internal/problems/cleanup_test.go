package problems

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
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
		other := Installed{
			Key: "optional-user", Enabled: true,
			Name: "Optional User", UniqueID: "Other.Optional",
			Dependencies: []manifest.Dependency{{UniqueID: "Tiles.Optional", Required: false}},
		}
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
		other := Installed{
			Key: "disabled-user", Enabled: false,
			Name: "Disabled User", UniqueID: "Other.Disabled",
			Dependencies: []manifest.Dependency{{UniqueID: "Tiles.Disabled", Required: false}},
		}
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
	return diskPack(t, id, map[string]string{"manifest.json": cpManifest(id), "content.json": content})
}

func modFolder(t *testing.T, name, file, content string) Installed {
	t.Helper()
	return diskPack(t, name, map[string]string{file: content})
}

func TestMapScanners(t *testing.T) {
	tmx := []byte(`<map><tileset name="a"><image source="../Maps/spring_town.png" width="1"/></tileset>` +
		`<tileset><image width="2" source='Tiles\Extra.png'/></tileset><imagelayer><image/></imagelayer></map>`)
	got := tmxImageSources(tmx)
	if len(got) != 2 || got[0] != "../Maps/spring_town.png" || got[1] != `Tiles\Extra.png` {
		t.Fatalf("tmx sources = %q", got)
	}
	runs := printableRuns([]byte("\x00\x01SharedTiles\x00ab\x00Maps/Town\x00SharedTiles"))
	if len(runs) != 2 || runs[0] != "sharedtiles" || runs[1] != "maps/town" {
		t.Fatalf("tbin runs = %q", runs)
	}
}

func TestRetextureOfVanillaSheetIsNotUnusedTilesheets(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	root := t.TempDir()
	writeProblemFile(t, root, "manifest.json", cpManifest("Colling.ElegantTools"))
	content := `{"Changes":[{"Action":"EditImage","Target":"TileSheets/tools","FromFile":"tools.png"}]}`
	if err := fsx.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	im := fromDisk(Installed{Key: "tools", Enabled: true, Folder: root, Name: "Elegant Tools", UniqueID: "Colling.ElegantTools"})
	if got := unusedTilesheetPacks([]Installed{im}); len(got) != 0 {
		t.Fatalf("a retexture is never an unused tilesheet pack, got %+v", got)
	}
}
