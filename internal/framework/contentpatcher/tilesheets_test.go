package contentpatcher

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestCleanupHintsTilesheets(t *testing.T) {
	t.Run("tmx use", func(t *testing.T) {
		pack := tilesheetCPPack(t, "Tiles.TMX", `{"Changes":[{"Action":"Load","Target":"Maps/SharedTiles"}]}`)
		other := cpModFolder(t, "Map User", "map.tmx", `<map><tileset><image source="Maps/SharedTiles.png"/></tileset></map>`)
		if got := cleanupsOf(pack, other); len(got) != 0 {
			t.Fatalf("TMX use was flagged: %#v", got)
		}
	})
	t.Run("tbin use", func(t *testing.T) {
		pack := tilesheetCPPack(t, "Tiles.TBIN", `{"Changes":[{"Action":"Load","Target":"TileSheets/SharedTiles"}]}`)
		other := cpModFolder(t, "Binary User", "map.tbin", "prefix sharedtiles suffix")
		if got := cleanupsOf(pack, other); len(got) != 0 {
			t.Fatalf("TBIN use was flagged: %#v", got)
		}
		if got := cleanupsOf(pack, other); len(got) != 0 {
			t.Fatalf("TBIN use was flagged from the kept match: %#v", got)
		}
	})
	t.Run("optional dependency", func(t *testing.T) {
		pack := tilesheetCPPack(t, "Tiles.Optional", `{"Changes":[{"Action":"Load","Target":"Maps/OptionalTiles"}]}`)
		other := cp.Mod{
			Key: "optional-user", Name: "Optional User", UniqueID: cp.SMAPI("Other.Optional"), Enabled: true,
			Dependencies: []cp.Dependency{{UniqueID: cp.SMAPI("Tiles.Optional")}},
		}
		if got := cleanupsOf(pack, other); len(got) != 0 {
			t.Fatalf("optional dependency was flagged: %#v", got)
		}
	})
	t.Run("unreferenced", func(t *testing.T) {
		pack := tilesheetCPPack(t, "Tiles.Unused", `{"Changes":[{"Action":"Load","Target":"Maps/UnusedTiles"}]}`)
		got := cleanupsOf(pack)
		if len(got) != 1 || got[0].Reason != "Tilesheets not used by any installed mod" {
			t.Fatalf("unreferenced tilesheet = %#v", got)
		}
	})
	t.Run("disabled use", func(t *testing.T) {
		pack := tilesheetCPPack(t, "Tiles.Disabled", `{"Changes":[{"Action":"Load","Target":"Maps/DisabledTiles"}]}`)
		other := cp.Mod{
			Key: "disabled-user", Name: "Disabled User", UniqueID: cp.SMAPI("Other.Disabled"), Enabled: false,
			Dependencies: []cp.Dependency{{UniqueID: cp.SMAPI("Tiles.Disabled")}},
		}
		got := cleanupsOf(pack, other)
		if len(got) != 1 || got[0].Reason != "Used only by switched-off mods: Disabled User" {
			t.Fatalf("disabled tilesheet = %#v", got)
		}
	})
	t.Run("edit data is not a tilesheet", func(t *testing.T) {
		pack := tilesheetCPPack(t, "Tiles.EditData", `{"Changes":[{"Action":"Load","Target":"Maps/EditDataTiles"},{"Action":"EditData","Target":"Data/Locations"}]}`)
		if got := cleanupsOf(pack); len(got) != 0 {
			t.Fatalf("EditData pack was flagged: %#v", got)
		}
	})
}

func tilesheetCPPack(t *testing.T, id, content string) cp.Mod {
	t.Helper()
	folder := t.TempDir()
	testfs.WriteFile(t, folder, "manifest.json", packs.Manifest(id))
	testfs.WriteFile(t, folder, "content.json", content)
	return cp.Mod{
		Key: id, Name: id, UniqueID: cp.SMAPI(id), Folder: folder, Enabled: true,
		ContentPackFor: cp.SMAPI("Pathoschild.ContentPatcher"),
	}
}

func cpModFolder(t *testing.T, name, file, content string) cp.Mod {
	t.Helper()
	folder := t.TempDir()
	testfs.WriteFile(t, folder, file, content)
	return cp.Mod{Key: name, Name: name, UniqueID: cp.SMAPI(name), Folder: folder, Enabled: true}
}

func TestRetextureOfVanillaSheetIsNotUnusedTilesheets(t *testing.T) {
	im := tilesheetCPPack(t, "Colling.ElegantTools", `{"Changes":[{"Action":"EditImage","Target":"TileSheets/tools","FromFile":"tools.png"}]}`)
	if got := cleanupsOf(im); len(got) != 0 {
		t.Fatalf("a retexture is never an unused tilesheet pack, got %+v", got)
	}
}

// cleanupsOf words the tilesheet cleanup of mods as one profile.
func cleanupsOf(mods ...cp.Mod) []framework.Cleanup {
	uses, unreadable := cp.TilesheetPacks(mods, false)
	return tilesheetCleanups(uses, unreadable)
}

// A pack that loads its own sheets supports other mods even when it also redraws them for a season or registers chair
// tiles; one that edits a map or only recolours existing sheets is a choice.
func TestSupportingPacks(t *testing.T) {
	for _, c := range []struct {
		id, content string
		want        bool
	}{
		{"Sheets.Own", `{"DynamicTokens":[{"Name":"Style","Value":"warm"}],"Changes":[{"Action":"Load","Target":"Maps/spring_own, Mods/own/extra"},{"Action":"EditImage","Target":"Maps/fall_own","FromFile":"fall.png"},{"Action":"EditData","Target":"Data/ChairTiles","Entries":{"own/0/0":"1/1/down/default/-1/true"}}]}`, true},
		{"Sheets.Recolour", `{"Changes":[{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"a.png"}]}`, false},
		{"Sheets.MapEdit", `{"Changes":[{"Action":"Load","Target":"Maps/own"},{"Action":"EditMap","Target":"Maps/Saloon","FromFile":"saloon.tmx"}]}`, false},
	} {
		if got := cp.IsSupportingPack(tilesheetCPPack(t, c.id, c.content), false); got != c.want {
			t.Errorf("%s: supporting = %v, want %v", c.id, got, c.want)
		}
	}
}
