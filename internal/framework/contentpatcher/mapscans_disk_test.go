package contentpatcher

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func TestMapScansAndListingsPersistAndPruneUninstalledFolders(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mapScans.Lock()
	mapScans.byPath = map[string]mapScan{
		"/mods/a/maps/x.tmx": {Size: 1, ModTime: 2, Keys: []string{"k"}},
		"/mods/gone/y.tbin":  {Size: 3, ModTime: 4, Runs: []string{"r"}},
	}
	mapScans.folders = map[string]mapFileList{
		"/mods/a":    {Stamp: "a", Paths: []string{"/mods/a/maps/x.tmx"}, Complete: true},
		"/mods/gone": {Stamp: "g", Paths: []string{"/mods/gone/y.tbin"}, Complete: true},
	}
	mapScans.loaded, mapScans.dirty = true, true
	mapScans.Unlock()

	flushMapScans([]framework.Mod{{Folder: "/mods/a"}})

	mapScans.Lock()
	resetMapScans()
	mapScans.Unlock()
	loadMapScans()

	mapScans.Lock()
	defer mapScans.Unlock()
	got, ok := mapScans.byPath["/mods/a/maps/x.tmx"]
	if len(mapScans.byPath) != 1 || !ok || got.Size != 1 || got.Keys[0] != "k" {
		t.Fatalf("reloaded scans = %#v", mapScans.byPath)
	}
	if list, ok := mapScans.folders["/mods/a"]; len(mapScans.folders) != 1 || !ok || list.Stamp != "a" || !list.Complete {
		t.Fatalf("reloaded folder listings = %#v", mapScans.folders)
	}
}
