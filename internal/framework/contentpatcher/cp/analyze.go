package cp

import (
	"maps"
	"slices"
)

// Analyze reads every enabled pack once; the tilesheet pass also covers the switched-off packs.
func Analyze(req Request) Findings {
	if req.CacheRoot != "" {
		cacheRoot = req.CacheRoot
	}
	key := analysisKey(req)
	if found, ok := lookupAnalysis(key); ok {
		return found
	}
	defer HoldPacks()()
	clearPackValidated()
	defer clearPackValidated()
	defer FlushMapScans(req.All)
	// After the tilesheet pass, so the packs it reads are written too and need no parse on the next check.
	defer FlushPackDiskCache(req.All)
	defer FlushParts()
	reads := recordReads()
	run := NewPartsRun()
	preloadContentPacks(req.Enabled, req.SkipImageOverlap)
	conflicts, conflictHints, shadowed, bundled := assetConflictScan(req.Enabled, run, req.SkipImageOverlap)
	var ext Extension
	if req.Extend != nil {
		ext = req.Extend(run, req.All)
	}
	tilesheets, unreadable := TilesheetPacks(req.All, req.SkipImageOverlap)
	found := Findings{
		Conflicts:            conflicts,
		Hints:                append(ext.Hints, conflictHints...),
		Shadowed:             shadowed,
		Bundled:              bundled,
		Mentioned:            mentionedIDs(req.Enabled, req.SkipImageOverlap),
		Supporting:           SupportingPackKeys(req.All, req.SkipImageOverlap),
		Tilesheets:           tilesheets,
		TilesheetsUnreadable: unreadable,
	}
	files, complete := reads()
	if complete {
		run.Commit()
	}
	if files, stamped := packStamps(req.All, files); complete && stamped {
		storeAnalysis(key, files, found)
	}
	return found
}

// Forget drops what the analysis keeps between checks.
func Forget() {
	packDiskState.Lock()
	packDiskState.loaded, packDiskState.entries, packDiskState.dirty = false, nil, false
	packDiskState.Unlock()
	mapScans.Lock()
	resetMapScans()
	mapScans.Unlock()
	DropParts()
	footprintMemo.Lock()
	footprintMemo.last = nil
	footprintMemo.Unlock()
}

// mentionedIDs are the mods the switched-on packs name in a HasMod condition.
func mentionedIDs(enabled []Mod, skip bool) []string {
	PreloadPacks(enabled, func(m Mod) Pack { return ReadPackForCleanup(m, skip) })
	set := map[string]bool{}
	for _, im := range enabled {
		maps.Copy(set, ReadPackForCleanup(im, skip).Mentions)
	}
	return slices.Sorted(maps.Keys(set))
}
