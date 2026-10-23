package contentpatcher

import (
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

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
	collections := func() int64 {
		var st debug.GCStats
		debug.ReadGCStats(&st)
		return st.NumGC
	}
	before := collections()
	releasePacks(gen)
	if _, held := packCache.Load(cleanFolder(im)); held {
		t.Fatal("packs stayed in memory after an idle release")
	}
	if collections() == before {
		t.Fatal("the released packs were left uncollected, so their pages stay with the process")
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

// A check that saved its disk cache leaves neither the parsed packs nor their encoded copies in memory once it ends.
func TestPacksLeaveMemoryWhenTheLastCheckEnds(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	packGrace = 0
	t.Cleanup(func() { packGrace = 30 * time.Second })
	im, _, _ := diskCachePack(t)
	end := holdPacks()
	readContentPack(im)
	flushPackDiskCache([]framework.Mod{im})
	end()
	if _, held := packCache.Load(cleanFolder(im)); held {
		t.Fatal("the parsed packs outlived the check")
	}
	packDiskState.Lock()
	defer packDiskState.Unlock()
	if packDiskState.entries != nil {
		t.Fatal("the encoded packs outlived the check")
	}
}

// The packs outlive a check by the grace period, so the next check of an edited profile finds them, and a check that
// starts inside it keeps them past the first release.
func TestPacksStayForTheGraceAfterACheck(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	packGrace = 100 * time.Millisecond
	t.Cleanup(func() { packGrace = 30 * time.Second })
	im, _, _ := diskCachePack(t)
	end := holdPacks()
	readContentPack(im)
	flushPackDiskCache([]framework.Mod{im})
	end()
	if _, held := packCache.Load(cleanFolder(im)); !held {
		t.Fatal("the packs left memory the moment the check ended")
	}
	next := holdPacks()
	time.Sleep(3 * packGrace)
	if _, held := packCache.Load(cleanFolder(im)); !held {
		t.Fatal("the packs were released under a running check")
	}
	next()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, held := packCache.Load(cleanFolder(im)); !held {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the packs stayed in memory after the grace")
}

func cleanFolder(im framework.Mod) string { return filepath.Clean(im.Folder) }
