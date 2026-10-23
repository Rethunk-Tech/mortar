package contentpatcher

import (
	"runtime/debug"
	"sync"
	"time"
)

// packGrace is how long the packs stay in memory after the last check ends, unless another check starts first.
var packGrace = 30 * time.Second

// A check reads every pack, then nothing reads them until the next one, which can be hours away; on a large profile
// they hold around 100 MB, while reading them back from the disk caches costs the next check a fraction of a second.
// So the packs leave memory once the last check holding them has been over for packGrace: a run of edits to one profile
// is a run of checks, and each would otherwise decode every pack again.

var packUse struct {
	sync.Mutex
	active int
	gen    int
}

// holdPacks marks a check that reads packs; the func it returns ends it and, once no check is running, releases them.
func holdPacks() func() {
	packUse.Lock()
	packUse.active++
	packUse.gen++
	packUse.Unlock()
	return func() {
		packUse.Lock()
		packUse.active--
		last, gen := packUse.active == 0, packUse.gen
		packUse.Unlock()
		switch {
		case !last:
		case packGrace <= 0:
			releasePacks(gen)
		default:
			time.AfterFunc(packGrace, func() { releasePacks(gen) })
		}
	}
}

// releasePacks drops the parsed packs and map scans when no check has run since gen, but only those the disk caches
// already hold: a cache with unsaved changes keeps its memory until a later check saves it.
func releasePacks(gen int) {
	if dropPacks(gen) {
		// The dropped packs are garbage nothing allocates enough to collect, and the pages they held stay with the
		// process until the runtime's slow scavenger returns them.
		debug.FreeOSMemory()
	}
}

func dropPacks(gen int) bool {
	packUse.Lock()
	defer packUse.Unlock()
	if packUse.active > 0 || packUse.gen != gen {
		return false
	}
	packDiskState.Lock()
	if !packDiskState.dirty && packDiskState.path != "" {
		packCache.Clear()
		packDiskState.loaded, packDiskState.entries = false, nil
	}
	packDiskState.Unlock()
	mapScans.Lock()
	if !mapScans.dirty {
		resetMapScans()
	}
	mapScans.Unlock()
	parts.Lock()
	if !parts.dirty {
		parts.entries, parts.loaded = map[string]partEntry{}, false
	}
	parts.Unlock()
	return true
}
