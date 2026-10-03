package problems

import (
	"bytes"
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
	basenamesByID := map[string][]string{}
	for id, assets := range assetsByID {
		basenamesByID[id] = assetBasenames(assets)
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
			for _, candidate := range candidates {
				id := strings.ToLower(candidate.UniqueID)
				if ext == ".tbin" {
					if slices.ContainsFunc(basenamesByID[id], func(name string) bool {
						return slices.ContainsFunc(scan.runs, func(run string) bool {
							return strings.Contains(run, name)
						})
					}) {
						recordTilesheetUse(uses[id], mod)
					}
					continue
				}
				if slices.ContainsFunc(scan.keys, func(key string) bool {
					return assetsByID[id][key]
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
	keys    []string
	runs    []string
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
	scan := mapScan{size: info.Size(), modTime: info.ModTime().UnixNano()}
	if ext == ".tbin" {
		scan.runs = printableRuns(raw)
	} else {
		for _, source := range tmxImageSources(raw) {
			normalized := normalizeAsset(source)
			scan.keys = append(scan.keys, normalized, filepath.Base(normalized))
		}
	}
	mapScans.Lock()
	mapScans.byPath[path] = scan
	mapScans.Unlock()
	return scan, nil
}

// tmxImageSources returns the source attribute of every <image> tag. A byte scan instead of
// encoding/xml: thousands of maps are read per check and only this one attribute matters.
func tmxImageSources(raw []byte) []string {
	var sources []string
	for rest := raw; ; {
		i := bytes.Index(rest, []byte("<image"))
		if i < 0 {
			return sources
		}
		rest = rest[i+len("<image"):]
		end := bytes.IndexByte(rest, '>')
		if end < 0 {
			return sources
		}
		tag := rest[:end]
		rest = rest[end:]
		j := bytes.Index(tag, []byte("source="))
		if j < 0 || j+len("source=") >= len(tag) {
			continue
		}
		value := tag[j+len("source="):]
		quote := value[0]
		if quote != '"' && quote != '\'' {
			continue
		}
		if k := bytes.IndexByte(value[1:], quote); k >= 0 {
			sources = append(sources, string(value[1:1+k]))
		}
	}
}

// printableRuns returns the distinct lower-cased runs of printable ASCII in a binary map,
// which is where a .tbin names its tilesheets; the tile data between them is dropped.
func printableRuns(raw []byte) []string {
	const minRun = 3
	seen := map[string]bool{}
	var runs []string
	start := -1
	flush := func(end int) {
		if start >= 0 && end-start >= minRun {
			run := strings.ToLower(string(raw[start:end]))
			if !seen[run] {
				seen[run] = true
				runs = append(runs, run)
			}
		}
		start = -1
	}
	for i, c := range raw {
		if c >= ' ' && c <= '~' {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(len(raw))
	return runs
}
