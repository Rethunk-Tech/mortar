package contentpatcher

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

// Once no check has run for a while the parsed packs leave memory, and the next read gets the same pack back from
// the disk cache; a cache with unsaved changes keeps them.
func TestIdlePacksAreReleasedAndReadBackFromDisk(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	im, _, _ := diskCachePack(t)
	first := readContentPack(im)
	packUse.Lock()
	gen := packUse.gen
	packUse.Unlock()
	releasePacks(gen)
	if _, held := packCache.Load(cleanFolder(im)); !held {
		t.Fatal("packs were released before the disk cache was saved")
	}
	flushPackDiskCache([]framework.Mod{im})
	releasePacks(gen)
	if _, held := packCache.Load(cleanFolder(im)); held {
		t.Fatal("packs stayed in memory after an idle release")
	}
	again := readContentPack(im)
	if len(again.patches) != len(first.patches) || again.fingerprint != first.fingerprint {
		t.Fatalf("pack read back = %#v, want %#v", again.patches, first.patches)
	}
	release := holdPacks()
	releasePacks(gen + 1)
	if _, held := packCache.Load(cleanFolder(im)); !held {
		t.Fatal("packs were released while a check held them")
	}
	release()
}

func cleanFolder(im framework.Mod) string { return filepath.Clean(im.Folder) }
