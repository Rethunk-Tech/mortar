package cp

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/go-safefs"
)

// MapFileList is one mod folder's .tmx and .tbin files for its store key. A store item never changes after it is
// extracted, so the list holds until the folder holds another key or is itself changed; the folder's own modification
// time catches files added or removed at its top level.
type MapFileList struct {
	Stamp    string   `json:"stamp"`
	Paths    []string `json:"paths,omitempty"`
	Complete bool     `json:"complete"`
}

// ModMapFiles lists the map files in mod's folder; complete is false when part of the folder could not be read.
func ModMapFiles(im Mod) (paths []string, complete bool) {
	stamp := im.Key
	info, err := os.Stat(im.Folder)
	if err == nil {
		stamp += "|" + strconv.FormatInt(info.ModTime().UnixNano(), 10)
	}
	noteRead(im.Folder, info)
	loadMapScans()
	mapScans.Lock()
	cached, ok := mapScans.folders[im.Folder]
	mapScans.Unlock()
	if ok && cached.Stamp == stamp {
		return cached.Paths, cached.Complete
	}
	unreadable := false
	if err := filepath.WalkDir(im.Folder, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return markUnreadable(&unreadable)
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !entry.IsDir() && (ext == ".tmx" || ext == ".tbin") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		unreadable = true
	}
	complete = !unreadable
	mapScans.Lock()
	mapScans.folders[im.Folder] = MapFileList{Stamp: stamp, Paths: paths, Complete: complete}
	mapScans.dirty = true
	mapScans.Unlock()
	return paths, complete
}

// markUnreadable records that a map could not be read, so the tilesheets it might use count as possibly used, and
// lets the walk go on to the other files.
func markUnreadable(flag *bool) error {
	*flag = true
	return nil
}

type tilesheetUse struct {
	enabled bool
	names   []string
}

// IsTilesheetPack is a Content Patcher pack whose every patch loads new map or tilesheet assets.
func IsTilesheetPack(im Mod, skip bool) bool {
	if !IsContentPatcherPack(im) {
		return false
	}
	pack := ReadPack(im, skip)
	if len(pack.Patches) == 0 {
		return false
	}
	named := false
	for _, patch := range pack.Patches {
		targets := SplitTargets(patch.Target)
		// Only a pack that loads new sheets can go unused: editing an existing sheet retextures
		// something the game always draws.
		if patch.Kind != "load" || len(targets) == 0 {
			return false
		}
		for _, target := range targets {
			if !IsTilesheetTarget(target) {
				return false
			}
			named = named || NormalizeAsset(target) != ""
		}
	}
	return named
}

// IsSupportingPack is a Content Patcher pack that supplies sheets for other mods: it loads at least one new map,
// tilesheet or mod asset, and otherwise only redraws images or registers chair tiles. A pack that edits a map or data,
// or only recolours existing sheets, is something the player chose.
func IsSupportingPack(im Mod, skip bool) bool {
	if !IsContentPatcherPack(im) {
		return false
	}
	sheet := func(target string) bool {
		target = NormalizeAsset(target)
		return IsTilesheetTarget(target) || strings.HasPrefix(target, "mods/")
	}
	loads := false
	for _, patch := range ReadPack(im, skip).Patches {
		switch {
		case patch.TokenName != "":
			// A dynamic token the pack defines, not a change.
		case patch.Kind == "load" && sheet(patch.Target):
			loads = true
		case patch.Action == kindEditImage:
		case patch.Target == "data/chairtiles":
		default:
			return false
		}
	}
	return loads
}

// SupportingPackKeys are the keys of the packs that supply sheets for other mods, switched on or off.
func SupportingPackKeys(mods []Mod, skip bool) []string {
	var out []string
	for _, im := range mods {
		if IsSupportingPack(im, skip) && !slices.Contains(out, im.Key) {
			out = append(out, im.Key)
		}
	}
	return out
}

// TilesheetPacks are the enabled packs that only load new map and tilesheet assets, and which mods use each one.
// Unreadable is true when a map could not be read, so any of them may be in use.
func TilesheetPacks(mods []Mod, skip bool) (uses []TilesheetUse, unreadable bool) {
	var candidates []Mod
	for _, im := range mods {
		if im.Enabled && IsTilesheetPack(im, skip) {
			candidates = append(candidates, im)
		}
	}
	if len(candidates) == 0 {
		return nil, false
	}

	used := make(map[string]*tilesheetUse, len(candidates))
	assetsByID := make(map[string]map[string]bool, len(candidates))
	for _, candidate := range candidates {
		id := candidate.UniqueID.Fold()
		used[id] = &tilesheetUse{}
		assets := map[string]bool{}
		for _, patch := range ReadPack(candidate, skip).Patches {
			for _, target := range SplitTargets(patch.Target) {
				AddAssetNames(assets, target)
			}
		}
		assetsByID[id] = assets
	}
	basenamesByID := map[string][]string{}
	for id, assets := range assetsByID {
		basenamesByID[id] = AssetBasenames(assets)
	}
	others := slices.DeleteFunc(slices.Clone(mods), func(im Mod) bool { return containsMod(candidates, im) })
	PreloadPacks(others, func(m Mod) Pack { return ReadPackForCleanup(m, skip) })
	for _, im := range others {
		mentions := ReadPackForCleanup(im, skip).Mentions
		for _, candidate := range candidates {
			id := candidate.UniqueID.Fold()
			if manifestUses(im, candidate.UniqueID) || mentions[id] {
				recordTilesheetUse(used[id], im)
			}
		}
	}

	for _, im := range mods {
		if containsMod(candidates, im) || im.Folder == "" {
			continue
		}
		if !RecordMapUses(im, candidates, assetsByID, basenamesByID, func(id ModID) {
			recordTilesheetUse(used[id.Fold()], im)
		}) {
			unreadable = true
		}
	}
	return tilesheetUses(candidates, used), unreadable
}

// RecordMapUses reports, through record, each candidate whose map or tilesheet assets a mod's map files load. It
// returns false when a map could not be read, since an unread map might use any candidate.
func RecordMapUses(im Mod, candidates []Mod, assetsByID map[string]map[string]bool, basenamesByID map[string][]string, record func(candidate ModID)) bool {
	readable := true
	paths, complete := ModMapFiles(im)
	if !complete {
		noteUnstampable()
		readable = false
	}
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			noteUnstampable()
			readable = false
			continue
		}
		noteRead(path, info)
		ext := strings.ToLower(filepath.Ext(path))
		scan, readErr := MapScanFor(path, fs.FileInfoToDirEntry(info), ext)
		if readErr != nil {
			noteUnstampable()
			readable = false
			continue
		}
		for _, candidate := range candidates {
			id := candidate.UniqueID.Fold()
			if ext == ".tbin" {
				if slices.ContainsFunc(basenamesByID[id], func(name string) bool { return tbinNames(path, scan, name) }) {
					record(candidate.UniqueID)
				}
				continue
			}
			if slices.ContainsFunc(scan.Keys, func(key string) bool {
				return assetsByID[id][key]
			}) {
				record(candidate.UniqueID)
			}
		}
	}
	return readable
}

// tilesheetUses reports, per candidate pack, whether a switched-on mod uses it and which switched-off mods do.
func tilesheetUses(candidates []Mod, uses map[string]*tilesheetUse) []TilesheetUse {
	out := make([]TilesheetUse, 0, len(candidates))
	for _, candidate := range candidates {
		use := uses[candidate.UniqueID.Fold()]
		names := slices.Clone(use.names)
		slices.Sort(names)
		out = append(out, TilesheetUse{
			Key: candidate.Key, ID: candidate.UniqueID, Name: candidate.Name, Enabled: use.enabled, Names: names,
		})
	}
	return out
}

// IsTilesheetTarget reports a target that names a map or tilesheet asset.
func IsTilesheetTarget(target string) bool {
	target = NormalizeAsset(target)
	return strings.HasPrefix(target, "maps/") || strings.HasPrefix(target, "tilesheets/")
}

// AddAssetNames records a target under its full name and its base name.
func AddAssetNames(assets map[string]bool, target string) {
	target = NormalizeAsset(target)
	if target == "" {
		return
	}
	assets[target] = true
	assets[filepath.Base(target)] = true
}

// NormalizeAsset is a target's asset name: lower-case, slash-separated, without a leading "./" or "..", and without
// its extension.
func NormalizeAsset(value string) string {
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

// AssetBasenames are the asset names in assets that are a bare file name.
func AssetBasenames(assets map[string]bool) []string {
	out := []string{}
	for asset := range assets {
		if asset == filepath.Base(asset) {
			out = append(out, asset)
		}
	}
	return out
}

func manifestUses(im Mod, uniqueID ModID) bool {
	for _, dep := range im.Dependencies {
		if Equal(dep.UniqueID, uniqueID) {
			return true
		}
	}
	return false
}

func recordTilesheetUse(use *tilesheetUse, im Mod) {
	if im.Enabled {
		use.enabled = true
		return
	}
	if !slices.Contains(use.names, im.Name) {
		use.names = append(use.names, im.Name)
	}
}

func containsMod(mods []Mod, want Mod) bool {
	return slices.ContainsFunc(mods, func(im Mod) bool { return im.Key == want.Key })
}

// MapScan is what the tilesheet scan needs from one map file, kept across checks until the
// file's size or modification time changes, since maps are many and rarely change.
type MapScan struct {
	Size    int64    `json:"size"`
	ModTime int64    `json:"modTime"`
	Keys    []string `json:"keys,omitempty"`
	Runs    []string `json:"runs,omitempty"`
	// Has holds the tilesheet names already looked for in Runs.
	Has map[string]bool `json:"has,omitempty"`
}

var mapScans = struct {
	sync.Mutex
	byPath  map[string]MapScan
	folders map[string]MapFileList
	loaded  bool
	dirty   bool
}{byPath: map[string]MapScan{}, folders: map[string]MapFileList{}}

func resetMapScans() {
	mapScans.byPath, mapScans.folders, mapScans.loaded, mapScans.dirty = map[string]MapScan{}, map[string]MapFileList{}, false, false
}

// tbinNames reports whether a tilesheet name occurs in the .tbin at path. Each answer is kept with the map's scan, as
// the same names are looked for in the same maps check after check.
func tbinNames(path string, scan MapScan, name string) bool {
	mapScans.Lock()
	found, known := scan.Has[name]
	mapScans.Unlock()
	if known {
		return found
	}
	found = slices.ContainsFunc(scan.Runs, func(run string) bool { return strings.Contains(run, name) })
	mapScans.Lock()
	defer mapScans.Unlock()
	if current, ok := mapScans.byPath[path]; ok && current.Size == scan.Size && current.ModTime == scan.ModTime {
		if current.Has == nil {
			current.Has = map[string]bool{}
			mapScans.byPath[path] = current
		}
		current.Has[name] = found
		mapScans.dirty = true
	}
	return found
}

// MapScanFor is what one map file holds, from the kept scan when its size and time still match.
func MapScanFor(path string, entry os.DirEntry, ext string) (MapScan, error) {
	info, err := entry.Info()
	if err != nil {
		return MapScan{}, err
	}
	loadMapScans()
	mapScans.Lock()
	cached, hit := mapScans.byPath[path]
	mapScans.Unlock()
	if hit && cached.Size == info.Size() && cached.ModTime == info.ModTime().UnixNano() {
		return cached, nil
	}
	raw, err := safefs.ReadFile(path)
	if err != nil {
		return MapScan{}, err
	}
	scan := MapScan{Size: info.Size(), ModTime: info.ModTime().UnixNano()}
	if ext == ".tbin" {
		scan.Runs = printableRuns(raw)
	} else {
		for _, source := range tmxImageSources(raw) {
			normalized := NormalizeAsset(source)
			scan.Keys = append(scan.Keys, normalized, filepath.Base(normalized))
		}
	}
	mapScans.Lock()
	mapScans.byPath[path] = scan
	mapScans.dirty = true
	mapScans.Unlock()
	return scan, nil
}

// TmxImageSources returns the source attribute of every <image> tag. A byte scan instead of
// encoding/xml: thousands of maps are read per check and only this one attribute matters.
func TmxImageSources(raw []byte) []string {
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

// tmxImageSources is TmxImageSources for the scan.
func tmxImageSources(raw []byte) []string { return TmxImageSources(raw) }

// PrintableRuns returns the distinct lower-cased runs of printable ASCII in a binary map,
// which is where a .tbin names its tilesheets; the tile data between them is dropped.
func PrintableRuns(raw []byte) []string {
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

func printableRuns(raw []byte) []string { return PrintableRuns(raw) }
