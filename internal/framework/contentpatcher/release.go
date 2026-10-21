package contentpatcher

import "sync"

// A check reads every pack, then nothing reads them until the next one, which can be hours away; on a large profile
// they hold around 100 MB, while reading them back from the disk caches costs the next check a fraction of a second.
// So the packs leave memory as soon as the last check holding them ends.

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
		if last {
			releasePacks(gen)
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
		resetMapScans()
	}
	mapScans.Unlock()
	parts.Lock()
	if !parts.dirty {
		parts.entries, parts.loaded = map[string]partEntry{}, false
	}
	parts.Unlock()
}
