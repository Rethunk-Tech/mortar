// Package contentpatcher is the Content Patcher framework: it reads the content packs written for it to find asset
// conflicts, shadowed packs, config settings that fight the profile's mods, and unused tilesheet packs.
package contentpatcher

import (
	"maps"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// ID is Content Patcher's mod id.
var ID = mod.SMAPI("Pathoschild.ContentPatcher")

var _ = framework.Register(Driver{})

// Driver is the Content Patcher framework.
type Driver struct{}

func (Driver) ID() mod.ID { return ID }

// IndexesAssets marks Content Patcher as the framework behind the asset map.
func (Driver) IndexesAssets() {}

// Matches is Content Patcher itself and every pack written for it.
func (Driver) Matches(m framework.Mod) bool {
	return mod.Equal(m.ModID(), ID) || isContentPatcherPack(m)
}

// Analyze reads every enabled pack once; tilesheet cleanup also covers disabled packs.
func (Driver) Analyze(in framework.Input) framework.Findings {
	key := analysisKey(in)
	if found, ok := lookupAnalysis(key); ok {
		return found
	}
	defer holdPacks()()
	clearPackValidated()
	defer clearPackValidated()
	defer flushMapScans(in.All)
	// After the cleanup pass, so the switched-off packs it reads are written too and need no parse on the next check.
	defer flushPackDiskCache(in.All)
	defer flushParts()
	reads := recordReads()
	run := newPartsRun()
	preloadContentPacks(in.Enabled, in.SkipImageOverlap)
	conflicts, conflictSettings, shadowed := assetConflictScan(in.Enabled, run, in.SkipImageOverlap)
	cleanup := unusedTilesheetPacks(in.All, in.SkipImageOverlap)
	found := framework.Findings{
		AssetConflicts: conflicts,
		Settings:       append(compatibilitySettings(in.Enabled, run, in.SkipImageOverlap), conflictSettings...),
		Redundant:      shadowed,
		Cleanup:        append(cleanup, recolourAddons(in.All)...),
		Supporting:     supportingPackKeys(in.All, in.SkipImageOverlap),
		Mentioned:      mentionedIDs(in.Enabled, in.SkipImageOverlap),
	}
	files, complete := reads()
	if complete {
		run.commit()
	}
	if files, stamped := packStamps(in.All, files); complete && stamped {
		storeAnalysis(key, files, found)
	}
	return found
}

// Forget drops what the driver keeps between checks.
func (Driver) Forget() {
	packDiskState.Lock()
	packDiskState.loaded, packDiskState.entries, packDiskState.dirty = false, nil, false
	packDiskState.Unlock()
	mapScans.Lock()
	resetMapScans()
	mapScans.Unlock()
	dropParts()
	footprintMemo.Lock()
	footprintMemo.last = nil
	footprintMemo.Unlock()
}

// mentionedIDs are the mods the switched-on packs name in a HasMod condition.
func mentionedIDs(enabled []framework.Mod, skip bool) []string {
	preloadPacks(enabled, func(m framework.Mod) cachedPack { return readContentPackForCleanup(m, skip) })
	set := map[string]bool{}
	for _, im := range enabled {
		maps.Copy(set, readContentPackForCleanup(im, skip).mentions)
	}
	return slices.Sorted(maps.Keys(set))
}
