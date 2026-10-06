package contentpatcher

import (
	"runtime/debug"
	"sync"
	"time"
)

// packIdleRelease is how long the parsed packs stay in memory after the last check. A check reads every pack, then
// nothing reads them until the next one, which can be hours away; on a large profile they hold around 100 MB, while
// reading them back from the disk caches costs the next check a fraction of a second.
const packIdleRelease = 2 * time.Minute

var packUse struct {
	sync.Mutex
	active int
	gen    int
}

// holdPacks marks a check that reads packs; the func it returns ends it and, once no check is running, schedules the
// packs' release.
func holdPacks() func() {
	packUse.Lock()
	packUse.active++
	packUse.gen++
	packUse.Unlock()
	return func() {
		packUse.Lock()
		defer packUse.Unlock()
		packUse.active--
		if packUse.active == 0 {
			gen := packUse.gen
			time.AfterFunc(packIdleRelease, func() { releasePacks(gen) })
		}
	}
}

// releasePacks drops the parsed packs and map scans when no check has run since gen, but only those the disk caches
// already hold: a cache with unsaved changes keeps its memory until a later check saves it.
func releasePacks(gen int) {
	packUse.Lock()
	defer packUse.Unlock()
	if packUse.active > 0 || packUse.gen != gen {
		return
	}
	packDiskState.Lock()
	if !packDiskState.dirty && packDiskState.path != "" {
		packCache.Clear()
		packDiskState.loaded, packDiskState.entries = false, nil
	}
	packDiskState.Unlock()
	mapScans.Lock()
	if !mapScans.dirty {
		mapScans.byPath, mapScans.loaded = map[string]mapScan{}, false
	}
	mapScans.Unlock()
	parts.Lock()
	if !parts.dirty {
		parts.entries, parts.loaded = map[string]partEntry{}, false
	}
	parts.Unlock()
	debug.FreeOSMemory()
}
