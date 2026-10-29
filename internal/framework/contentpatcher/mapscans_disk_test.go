package contentpatcher

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestMapScansAndListingsPersistAndPruneUninstalledFolders(t *testing.T) {
	testfs.DataHome(t)
	mapScans.Lock()
	mapScans.byPath = map[string]mapScan{
		filepath.FromSlash("/mods/a/maps/x.tmx"): {Size: 1, ModTime: 2, Keys: []string{"k"}},
		filepath.FromSlash("/mods/gone/y.tbin"):  {Size: 3, ModTime: 4, Runs: []string{"r"}},
	}
	mapScans.folders = map[string]mapFileList{
		filepath.FromSlash("/mods/a"):    {Stamp: "a", Paths: []string{filepath.FromSlash("/mods/a/maps/x.tmx")}, Complete: true},
		filepath.FromSlash("/mods/gone"): {Stamp: "g", Paths: []string{filepath.FromSlash("/mods/gone/y.tbin")}, Complete: true},
	}
	mapScans.loaded, mapScans.dirty = true, true
	mapScans.Unlock()

	flushMapScans([]framework.Mod{{Folder: filepath.FromSlash("/mods/a")}})

	mapScans.Lock()
	resetMapScans()
	mapScans.Unlock()
	loadMapScans()

	mapScans.Lock()
	defer mapScans.Unlock()
	got, ok := mapScans.byPath[filepath.FromSlash("/mods/a/maps/x.tmx")]
	if len(mapScans.byPath) != 1 || !ok || got.Size != 1 || got.Keys[0] != "k" {
		t.Fatalf("reloaded scans = %#v", mapScans.byPath)
	}
	if list, ok := mapScans.folders[filepath.FromSlash("/mods/a")]; len(mapScans.folders) != 1 || !ok || list.Stamp != "a" || !list.Complete {
		t.Fatalf("reloaded folder listings = %#v", mapScans.folders)
	}
}
