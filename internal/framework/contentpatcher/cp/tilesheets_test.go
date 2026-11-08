package cp

import (
	"path/filepath"
	"testing"
)

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

// A TBIN map's name matches are kept with its scan, so the next check reads them back instead of rescanning.
func TestTbinNameMatchIsKeptWithTheScan(t *testing.T) {
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)

	pack := contentPack(t, "Tiles.TBIN", `{"Changes":[{"Action":"Load","Target":"TileSheets/SharedTiles"}]}`)
	other := diskPack(t, "Binary User", map[string]string{"map.tbin": "prefix sharedtiles suffix"})
	if got := check([]Mod{pack, other}).Tilesheets; len(got) != 1 || !got[0].Enabled {
		t.Fatalf("TBIN use was flagged: %#v", got)
	}
	mapScans.Lock()
	scan := mapScans.byPath[filepath.Join(other.Folder, "map.tbin")]
	mapScans.Unlock()
	if !scan.Has["sharedtiles"] {
		t.Fatalf("the name match was not kept with the scan: %#v", scan)
	}
	if got := check([]Mod{pack, other}).Tilesheets; len(got) != 1 || !got[0].Enabled {
		t.Fatalf("TBIN use was flagged from the kept match: %#v", got)
	}
}
