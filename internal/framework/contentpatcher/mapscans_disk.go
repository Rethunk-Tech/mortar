package contentpatcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// mapScanCacheVersion changes whenever what mapScan records changes, so an older file is ignored.
const mapScanCacheVersion = 2

type diskMapScans struct {
	Version int                    `json:"version"`
	Scans   map[string]mapScan     `json:"scans"`
	Folders map[string]mapFileList `json:"folders"`
}

func mapScanCachePath() (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache", "problems-map-scans.json"), nil
}

// loadMapScans reads the saved scans once per process. Each entry is still checked against its file's size and
// modification time when used, so a stale entry costs a rescan and nothing more.
func loadMapScans() {
	mapScans.Lock()
	defer mapScans.Unlock()
	if mapScans.loaded {
		return
	}
	mapScans.loaded = true
	path, err := mapScanCachePath()
	if err != nil {
		return
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return
	}
	payload, ok := unzipPackCache(raw)
	if !ok {
		return
	}
	var saved diskMapScans
	if json.Unmarshal(payload, &saved) == nil && saved.Version == mapScanCacheVersion && saved.Scans != nil && saved.Folders != nil {
		mapScans.byPath, mapScans.folders = saved.Scans, saved.Folders
	}
}

// flushMapScans drops the scans and listings of folders that are no longer installed, then saves them when any
// changed.
func flushMapScans(mods []framework.Mod) {
	prefixes := make([]string, 0, len(mods))
	installed := map[string]bool{}
	for _, im := range mods {
		if im.Folder != "" {
			prefixes = append(prefixes, filepath.Clean(im.Folder)+string(filepath.Separator))
			installed[im.Folder] = true
		}
	}
	mapScans.Lock()
	defer mapScans.Unlock()
	for path := range mapScans.byPath {
		keep := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) {
				keep = true
				break
			}
		}
		if !keep {
			delete(mapScans.byPath, path)
			mapScans.dirty = true
		}
	}
	for folder := range mapScans.folders {
		if !installed[folder] {
			delete(mapScans.folders, folder)
			mapScans.dirty = true
		}
	}
	if !mapScans.dirty {
		return
	}
	path, err := mapScanCachePath()
	if err != nil {
		return
	}
	raw, err := json.Marshal(diskMapScans{Version: mapScanCacheVersion, Scans: mapScans.byPath, Folders: mapScans.folders})
	if err != nil {
		return
	}
	packed, err := zipPackCache(raw)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil || datadir.WriteFile(path, packed, 0o600) != nil {
		return
	}
	mapScans.dirty = false
}
