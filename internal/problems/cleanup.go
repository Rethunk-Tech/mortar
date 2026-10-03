package problems

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

// markUnreadable records that a map could not be read, so the tilesheets it might use count as possibly used, and
// lets the walk go on to the other files.
func markUnreadable(flag *bool) error {
	*flag = true
	return nil
}

func cleanupHints(mods []Installed) []Cleanup {
	enabledNeeds := map[string]bool{}
	disabledDependents := map[string]bool{}
	tilesheets := unusedTilesheetPacks(mods)
	tilesheetIDs := map[string]bool{}
	for _, tilesheet := range tilesheets {
		tilesheetIDs[strings.ToLower(strings.TrimSpace(tilesheet.UniqueID))] = true
	}
	for _, mod := range mods {
		for _, dep := range mod.Dependencies {
			recordCleanupDependency(mod, dep.UniqueID, enabledNeeds, disabledDependents)
		}
		if mod.ContentPackFor != "" {
			recordCleanupDependency(mod, mod.ContentPackFor, enabledNeeds, disabledDependents)
		}
	}

	out := make([]Cleanup, 0)
	for _, mod := range mods {
		id := strings.ToLower(strings.TrimSpace(mod.UniqueID))
		if id == "" || mod.ContentPackFor != "" || tilesheetIDs[id] || enabledNeeds[id] || !disabledDependents[id] {
			continue
		}
		out = append(out, Cleanup{Key: mod.Key, UniqueID: mod.UniqueID, Name: mod.Name})
	}
	out = append(out, tilesheets...)
	slices.SortFunc(out, func(a, b Cleanup) int {
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.UniqueID), strings.ToLower(b.UniqueID))
	})
	return out
}

func recordCleanupDependency(mod Installed, uniqueID string, enabledNeeds, disabledDependents map[string]bool) {
	id := strings.ToLower(strings.TrimSpace(uniqueID))
	if id == "" {
		return
	}
	if mod.Enabled {
		enabledNeeds[id] = true
	} else {
		disabledDependents[id] = true
	}
}

type tilesheetUse struct {
	enabled bool
	names   []string
}

func unusedTilesheetPacks(mods []Installed) []Cleanup {
	var candidates []Installed
	for _, mod := range mods {
		if !mod.Enabled || !isContentPatcherPack(mod.Folder) {
			continue
		}
		pack := readContentPack(mod)
		if len(pack.patches) == 0 {
			continue
		}
		assets := map[string]bool{}
		eligible := true
		for _, patch := range pack.patches {
			targets := splitTargets(patch.target)
			if patch.kind == "other" || len(targets) == 0 ||
				(patch.kind != "load" && !patch.image) {
				eligible = false
				break
			}
			for _, target := range targets {
				if !isTilesheetTarget(target) {
					eligible = false
					break
				}
				addAssetNames(assets, target)
			}
			if !eligible {
				break
			}
		}
		if eligible && len(assets) > 0 {
			candidates = append(candidates, mod)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	uses := make(map[string]*tilesheetUse, len(candidates))
	assetsByID := make(map[string]map[string]bool, len(candidates))
	for _, candidate := range candidates {
		id := strings.ToLower(candidate.UniqueID)
		uses[id] = &tilesheetUse{}
		assets := map[string]bool{}
		for _, patch := range readContentPack(candidate).patches {
			for _, target := range splitTargets(patch.target) {
				addAssetNames(assets, target)
			}
		}
		assetsByID[id] = assets
	}
	for _, mod := range mods {
		if containsMod(candidates, mod) {
			continue
		}
		for _, candidate := range candidates {
			if manifestUses(mod, candidate.UniqueID) || contentMentions(mod, candidate.UniqueID) {
				recordTilesheetUse(uses[strings.ToLower(candidate.UniqueID)], mod)
			}
		}
	}

	mapsUnreadable := false
	for _, mod := range mods {
		if containsMod(candidates, mod) || mod.Folder == "" {
			continue
		}
		if err := filepath.WalkDir(mod.Folder, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return markUnreadable(&mapsUnreadable)
			}
			if entry.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".tmx" && ext != ".tbin" {
				return nil
			}
			scan, readErr := mapScanFor(path, entry, ext)
			if readErr != nil {
				return markUnreadable(&mapsUnreadable)
			}
			if !scan.ok {
				mapsUnreadable = true
				return nil
			}
			for _, candidate := range candidates {
				id := strings.ToLower(candidate.UniqueID)
				if ext == ".tbin" {
					if slices.ContainsFunc(assetBasenames(assetsByID[id]), func(name string) bool {
						return bytes.Contains(scan.lower, []byte(name))
					}) {
						recordTilesheetUse(uses[id], mod)
					}
					continue
				}
				if slices.ContainsFunc(scan.sources, func(source string) bool {
					return assetMatches(assetsByID[id], source)
				}) {
					recordTilesheetUse(uses[id], mod)
				}
			}
			return nil
		}); err != nil {
			mapsUnreadable = true
		}
	}
	out := []Cleanup{}
	for _, candidate := range candidates {
		use := uses[strings.ToLower(candidate.UniqueID)]
		if mapsUnreadable || use.enabled {
			continue
		}
		reason := "Tilesheets not used by any installed mod"
		if len(use.names) > 0 {
			slices.Sort(use.names)
			reason = "Used only by switched-off mods: " + strings.Join(use.names, ", ")
		}
		out = append(out, Cleanup{
			Key: candidate.Key, UniqueID: candidate.UniqueID, Name: candidate.Name, Reason: reason,
		})
	}
	return out
}

func isTilesheetTarget(target string) bool {
	target = normalizeAsset(target)
	return strings.HasPrefix(target, "maps/") || strings.HasPrefix(target, "tilesheets/")
}

func addAssetNames(assets map[string]bool, target string) {
	target = normalizeAsset(target)
	if target == "" {
		return
	}
	assets[target] = true
	assets[filepath.Base(target)] = true
}

func normalizeAsset(value string) string {
	value = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "\\", "/")))
	value = strings.TrimPrefix(value, "./")
	for strings.HasPrefix(value, "../") {
		value = strings.TrimPrefix(value, "../")
	}
	if ext := filepath.Ext(value); ext != "" {
		value = strings.TrimSuffix(value, ext)
	}
	return value
}

func assetBasenames(assets map[string]bool) []string {
	out := []string{}
	for asset := range assets {
		if asset == filepath.Base(asset) {
			out = append(out, asset)
		}
	}
	return out
}

func assetMatches(assets map[string]bool, source string) bool {
	normalized := normalizeAsset(source)
	return assets[normalized] || assets[filepath.Base(normalized)]
}

func manifestUses(mod Installed, uniqueID string) bool {
	return slices.ContainsFunc(mod.Dependencies, func(dep manifest.Dependency) bool {
		return sameID(dep.UniqueID, uniqueID)
	})
}

func contentMentions(mod Installed, uniqueID string) bool {
	return readContentPackForCleanup(mod).mentions[strings.ToLower(uniqueID)]
}

func recordTilesheetUse(use *tilesheetUse, mod Installed) {
	if mod.Enabled {
		use.enabled = true
		return
	}
	if !slices.Contains(use.names, mod.Name) {
		use.names = append(use.names, mod.Name)
	}
}

func containsMod(mods []Installed, want Installed) bool {
	return slices.ContainsFunc(mods, func(mod Installed) bool { return mod.Key == want.Key })
}

// mapScan is what the tilesheet scan needs from one map file, kept across checks until the
// file's size or modification time changes, since maps are many and rarely change.
type mapScan struct {
	size    int64
	modTime int64
	sources []string
	lower   []byte
	ok      bool
}

var mapScans = struct {
	sync.Mutex
	byPath map[string]mapScan
}{byPath: map[string]mapScan{}}

func mapScanFor(path string, entry os.DirEntry, ext string) (mapScan, error) {
	info, err := entry.Info()
	if err != nil {
		return mapScan{}, err
	}
	mapScans.Lock()
	cached, hit := mapScans.byPath[path]
	mapScans.Unlock()
	if hit && cached.size == info.Size() && cached.modTime == info.ModTime().UnixNano() {
		return cached, nil
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return mapScan{}, err
	}
	scan := mapScan{size: info.Size(), modTime: info.ModTime().UnixNano(), ok: true}
	if ext == ".tbin" {
		scan.lower = bytes.ToLower(raw)
	} else {
		scan.sources, scan.ok = tmxImageSources(raw)
	}
	mapScans.Lock()
	mapScans.byPath[path] = scan
	mapScans.Unlock()
	return scan, nil
}

func tmxImageSources(raw []byte) ([]string, bool) {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	var sources []string
	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return sources, true
			}
			return nil, false
		}
		start, ok := token.(xml.StartElement)
		if !ok || !strings.EqualFold(start.Name.Local, "image") {
			continue
		}
		for _, attr := range start.Attr {
			if strings.EqualFold(attr.Name.Local, "source") {
				sources = append(sources, attr.Value)
			}
		}
	}
}
