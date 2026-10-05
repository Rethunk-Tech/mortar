package contentpatcher

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

// markUnreadable records that a map could not be read, so the tilesheets it might use count as possibly used, and
// lets the walk go on to the other files.
// mapFiles remembers each mod folder's .tmx and .tbin files for its store key. A store item never changes after it
// is extracted, so the list holds until the folder holds another key or is itself changed; the folder's own
// modification time catches files added or removed at its top level.
var mapFiles = struct {
	sync.Mutex
	byFolder map[string]mapFileList
}{byFolder: map[string]mapFileList{}}

type mapFileList struct {
	stamp    string
	paths    []string
	complete bool
}

// modMapFiles lists the map files in mod's folder; complete is false when part of the folder could not be read.
func modMapFiles(im framework.Mod) (paths []string, complete bool) {
	stamp := im.Key
	if info, err := os.Stat(im.Folder); err == nil {
		stamp += "|" + strconv.FormatInt(info.ModTime().UnixNano(), 10)
	}
	mapFiles.Lock()
	cached, ok := mapFiles.byFolder[im.Folder]
	mapFiles.Unlock()
	if ok && cached.stamp == stamp {
		return cached.paths, cached.complete
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
	mapFiles.Lock()
	mapFiles.byFolder[im.Folder] = mapFileList{stamp: stamp, paths: paths, complete: complete}
	mapFiles.Unlock()
	return paths, complete
}

func markUnreadable(flag *bool) error {
	*flag = true
	return nil
}

type tilesheetUse struct {
	enabled bool
	names   []string
}

func unusedTilesheetPacks(mods []framework.Mod) []framework.Cleanup {
	var candidates []framework.Mod
	for _, im := range mods {
		if !im.Enabled || !isContentPatcherPack(im) {
			continue
		}
		pack := readContentPack(im)
		if len(pack.patches) == 0 {
			continue
		}
		assets := map[string]bool{}
		eligible := true
		for _, patch := range pack.patches {
			targets := splitTargets(patch.target)
			// Only a pack that loads new sheets can go unused: editing an existing sheet retextures
			// something the game always draws.
			if patch.kind != "load" || len(targets) == 0 {
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
			candidates = append(candidates, im)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	uses := make(map[string]*tilesheetUse, len(candidates))
	assetsByID := make(map[string]map[string]bool, len(candidates))
	for _, candidate := range candidates {
		id := candidate.ModID().Fold()
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
	for _, im := range mods {
		if containsMod(candidates, im) {
			continue
		}
		mentions := readContentPackForCleanup(im).mentions
		for _, candidate := range candidates {
			id := candidate.ModID().Fold()
			if manifestUses(im, candidate.ModID()) || mentions[id] {
				recordTilesheetUse(uses[id], im)
			}
		}
	}

	mapsUnreadable := false
	for _, im := range mods {
		if containsMod(candidates, im) || im.Folder == "" {
			continue
		}
		paths, complete := modMapFiles(im)
		if !complete {
			mapsUnreadable = true
		}
		for _, path := range paths {
			info, statErr := os.Stat(path)
			if statErr != nil {
				mapsUnreadable = true
				continue
			}
			ext := strings.ToLower(filepath.Ext(path))
			scan, readErr := mapScanFor(path, fs.FileInfoToDirEntry(info), ext)
			if readErr != nil {
				mapsUnreadable = true
				continue
			}
			for _, candidate := range candidates {
				id := candidate.ModID().Fold()
				if ext == ".tbin" {
					if slices.ContainsFunc(basenamesByID[id], func(name string) bool {
						return slices.ContainsFunc(scan.Runs, func(run string) bool {
							return strings.Contains(run, name)
						})
					}) {
						recordTilesheetUse(uses[id], im)
					}
					continue
				}
				if slices.ContainsFunc(scan.Keys, func(key string) bool {
					return assetsByID[id][key]
				}) {
					recordTilesheetUse(uses[id], im)
				}
			}
		}
	}
	out := []framework.Cleanup{}
	for _, candidate := range candidates {
		use := uses[candidate.ModID().Fold()]
		if mapsUnreadable || use.enabled {
			continue
		}
		reason := "Tilesheets not used by any installed mod"
		if len(use.names) > 0 {
			slices.Sort(use.names)
			reason = "Used only by switched-off mods: " + strings.Join(use.names, ", ")
		}
		out = append(out, framework.Cleanup{
			Key: candidate.Key, ID: candidate.ModID(), Name: candidate.Name, Reason: reason,
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

func manifestUses(im framework.Mod, uniqueID mod.ID) bool {
	return slices.ContainsFunc(im.Dependencies, func(dep manifest.Dependency) bool {
		return mod.Equal(dep.ModID(), uniqueID)
	})
}

func recordTilesheetUse(use *tilesheetUse, im framework.Mod) {
	if im.Enabled {
		use.enabled = true
		return
	}
	if !slices.Contains(use.names, im.Name) {
		use.names = append(use.names, im.Name)
	}
}

func containsMod(mods []framework.Mod, want framework.Mod) bool {
	return slices.ContainsFunc(mods, func(im framework.Mod) bool { return im.Key == want.Key })
}

// mapScan is what the tilesheet scan needs from one map file, kept across checks until the
// file's size or modification time changes, since maps are many and rarely change.
type mapScan struct {
	Size    int64    `json:"size"`
	ModTime int64    `json:"modTime"`
	Keys    []string `json:"keys,omitempty"`
	Runs    []string `json:"runs,omitempty"`
}

var mapScans = struct {
	sync.Mutex
	byPath map[string]mapScan
	loaded bool
	dirty  bool
}{byPath: map[string]mapScan{}}

func mapScanFor(path string, entry os.DirEntry, ext string) (mapScan, error) {
	info, err := entry.Info()
	if err != nil {
		return mapScan{}, err
	}
	loadMapScans()
	mapScans.Lock()
	cached, hit := mapScans.byPath[path]
	mapScans.Unlock()
	if hit && cached.Size == info.Size() && cached.ModTime == info.ModTime().UnixNano() {
		return cached, nil
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return mapScan{}, err
	}
	scan := mapScan{Size: info.Size(), ModTime: info.ModTime().UnixNano()}
	if ext == ".tbin" {
		scan.Runs = printableRuns(raw)
	} else {
		for _, source := range tmxImageSources(raw) {
			normalized := normalizeAsset(source)
			scan.Keys = append(scan.Keys, normalized, filepath.Base(normalized))
		}
	}
	mapScans.Lock()
	mapScans.byPath[path] = scan
	mapScans.dirty = true
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
