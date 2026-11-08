package cp

import (
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"
)

// Once no check has run for a while the parsed packs leave memory, and the next read gets the same pack back from
// the disk cache; a cache with unsaved changes keeps them.
func TestIdlePacksAreReleasedAndReadBackFromDisk(t *testing.T) {
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)
	im, _, _ := diskCachePack(t)
	first := ReadPack(im, false)
	packUse.Lock()
	gen := packUse.gen
	packUse.Unlock()
	ReleasePacks(gen)
	if _, held := packCache.Load(cleanFolder(im)); !held {
		t.Fatal("packs were released before the disk cache was saved")
	}
	FlushPackDiskCache([]Mod{im})
	collections := func() int64 {
		var st debug.GCStats
		debug.ReadGCStats(&st)
		return st.NumGC
	}
	before := collections()
	ReleasePacks(gen)
	if _, held := packCache.Load(cleanFolder(im)); held {
		t.Fatal("packs stayed in memory after an idle release")
	}
	if collections() == before {
		t.Fatal("the released packs were left uncollected, so their pages stay with the process")
	}
	again := ReadPack(im, false)
	if len(again.Patches) != len(first.Patches) || again.Fingerprint != first.Fingerprint {
		t.Fatalf("pack read back = %#v, want %#v", again.Patches, first.Patches)
	}
	release := HoldPacks()
	ReleasePacks(gen + 1)
	if _, held := packCache.Load(cleanFolder(im)); !held {
		t.Fatal("packs were released while a check held them")
	}
	release()
}

// A check that saved its disk cache leaves neither the parsed packs nor their encoded copies in memory once it ends.
func TestPacksLeaveMemoryWhenTheLastCheckEnds(t *testing.T) {
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)
	packGrace = 0
	t.Cleanup(func() { packGrace = 30 * time.Second })
	im, _, _ := diskCachePack(t)
	end := HoldPacks()
	ReadPack(im, false)
	FlushPackDiskCache([]Mod{im})
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
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)
	// No grace timer fires while the first checks look, however slow the runner (the race detector on CI).
	packGrace = time.Hour
	t.Cleanup(func() { packGrace = 30 * time.Second })
	im, _, _ := diskCachePack(t)
	end := HoldPacks()
	ReadPack(im, false)
	FlushPackDiskCache([]Mod{im})
	packUse.Lock()
	ended := packUse.gen
	packUse.Unlock()
	end()
	if _, held := packCache.Load(cleanFolder(im)); !held {
		t.Fatal("the packs left memory the moment the check ended")
	}
	next := HoldPacks()
	// The first check's grace running out under a newer check must not release the packs.
	ReleasePacks(ended)
	if _, held := packCache.Load(cleanFolder(im)); !held {
		t.Fatal("the packs were released under a running check")
	}
	packGrace = 40 * time.Millisecond
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

func cleanFolder(im Mod) string { return filepath.Clean(im.Folder) }
