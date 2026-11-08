package cp

import (
	"errors"
	"sync"
)

var (
	cacheRoot      string
	errNoCacheRoot = errors.New("no cache root")
)

// SetCacheRoot is where the on-disk caches live, normally the app's data folder. Set it before the first read; a
// pack is keyed relative to it, so a moved data folder keeps its entries.
func SetCacheRoot(dir string) { cacheRoot = dir }

// Reset drops every in-memory cache, for a caller that needs a clean slate.
func Reset() {
	packCache = sync.Map{}
	pngShapeCache = sync.Map{}
	pngAlphaCache = sync.Map{}
	cellSetCache = sync.Map{}
	mapCache = sync.Map{}
	packDiskState.Lock()
	packDiskState.loaded, packDiskState.path, packDiskState.base = false, "", ""
	packDiskState.entries, packDiskState.dirty = nil, false
	packDiskState.Unlock()
}
