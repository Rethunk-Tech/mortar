package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const (
	historyFilesDir      = "history-files"
	historyBlobsDir      = "blobs"
	historySnapshotIndex = "index.json"
)

func captureHistoryConfigs(dir, snapshotID string, entries []Entry) {
	if snapshotID == "" {
		return
	}
	idx := map[string]map[string]string{}
	mods := filepath.Join(dir, "mods")
	for _, e := range entries {
		files := configFilesIn(liveEntryDir(mods, e.Key))
		if len(files) == 0 {
			continue
		}
		paths := make(map[string]string, len(files))
		for rel, body := range files {
			sum := sha256.Sum256(body)
			hash := hex.EncodeToString(sum[:])
			blob := filepath.Join(dir, historyFilesDir, historyBlobsDir, hash)
			if _, err := os.Stat(blob); errors.Is(err, os.ErrNotExist) {
				if err := os.MkdirAll(filepath.Dir(blob), 0o700); err != nil {
					return
				}
				if err := datadir.WriteFile(blob, body, 0o600); err != nil {
					return
				}
			} else if err != nil {
				return
			}
			paths[rel] = hash
		}
		idx[e.Key] = paths
	}
	snapDir := filepath.Join(dir, historyFilesDir, snapshotID)
	_ = os.RemoveAll(snapDir)
	if err := os.MkdirAll(snapDir, 0o700); err != nil {
		return
	}
	if err := datadir.WriteJSON(filepath.Join(snapDir, historySnapshotIndex), idx); err != nil {
		return
	}
}

func loadHistoryConfigs(dir, snapshotID string, entries []Entry) map[string]map[string][]byte {
	out := emptyConfigs()
	if snapshotID == "" {
		return out
	}
	idx, err := readSnapshotIndex(dir, snapshotID)
	if err != nil {
		return out
	}
	for _, e := range entries {
		files := idx[e.Key]
		if len(files) == 0 {
			continue
		}
		loaded := make(map[string][]byte, len(files))
		for rel, hash := range files {
			body, err := fsx.ReadFile(filepath.Join(dir, historyFilesDir, historyBlobsDir, hash))
			if err != nil {
				continue
			}
			loaded[rel] = body
		}
		if len(loaded) > 0 {
			out[entryIdentity(e)] = loaded
		}
	}
	return out
}

func readLiveConfigs(mods string, entries []Entry) map[string]map[string][]byte {
	out := emptyConfigs()
	for _, e := range entries {
		files := configFilesIn(liveEntryDir(mods, e.Key))
		if len(files) > 0 {
			out[entryIdentity(e)] = files
		}
	}
	return out
}

func configFilesIn(root string) map[string][]byte {
	files := map[string][]byte{}
	addConfigFile(files, root, filepath.Join(root, "config.json"))
	return files
}

func addConfigFile(files map[string][]byte, root, path string) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return
	}
	slash := filepath.ToSlash(rel)
	if !datadir.WritableRel(slash) {
		return
	}
	if body, err := fsx.ReadFile(path); err == nil {
		files[slash] = body
	}
}

func restoreHistoryConfig(dir, snapshotID, key, file string) error {
	idx, err := readSnapshotIndex(dir, snapshotID)
	if err != nil {
		return err
	}
	hash, ok := idx[key][file]
	if !ok || hash == "" {
		return fmt.Errorf("history config %s/%s not found", key, file)
	}
	body, err := fsx.ReadFile(filepath.Join(dir, historyFilesDir, historyBlobsDir, hash))
	if err != nil {
		return err
	}
	dst := filepath.Join(liveEntryDir(filepath.Join(dir, "mods"), key), filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return datadir.WriteFile(dst, body, 0o600)
}

func readSnapshotIndex(dir, snapshotID string) (map[string]map[string]string, error) {
	raw, err := fsx.ReadFile(filepath.Join(dir, historyFilesDir, snapshotID, historySnapshotIndex))
	if err != nil {
		return nil, err
	}
	var idx map[string]map[string]string
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, err
	}
	if idx == nil {
		idx = map[string]map[string]string{}
	}
	return idx, nil
}

func pruneHistoryFiles(dir string, referenced map[string]struct{}) {
	root := filepath.Join(dir, historyFilesDir)
	ents, err := os.ReadDir(root)
	if err != nil {
		return
	}
	keepBlobs := map[string]struct{}{}
	for _, ent := range ents {
		name := ent.Name()
		if name == historyBlobsDir {
			continue
		}
		path := filepath.Join(root, name)
		if _, ok := referenced[name]; !ok {
			_ = os.RemoveAll(path)
			continue
		}
		idx, err := readSnapshotIndex(dir, name)
		if err != nil {
			_ = os.RemoveAll(path)
			continue
		}
		for _, files := range idx {
			for _, hash := range files {
				if hash != "" {
					keepBlobs[hash] = struct{}{}
				}
			}
		}
	}
	blobs, err := os.ReadDir(filepath.Join(root, historyBlobsDir))
	if err != nil {
		return
	}
	for _, ent := range blobs {
		if _, ok := keepBlobs[ent.Name()]; ok {
			continue
		}
		_ = os.RemoveAll(filepath.Join(root, historyBlobsDir, ent.Name()))
	}
}

func predecessorOf(data historyFileData, eventID string) (before []Entry, beforeID string, event HistoryEvent, ok bool) {
	for i, ev := range data.Events {
		if ev.ID != eventID {
			continue
		}
		if i > 0 {
			prev := data.Events[i-1]
			before, ok = snapshotEntries(&data, prev.SnapshotID)
			beforeID = prev.SnapshotID
			if !ok {
				before = nil
				beforeID = ""
			}
		}
		return before, beforeID, ev, true
	}
	return nil, "", HistoryEvent{}, false
}

func itemMatches(it HistoryItem, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	if strings.EqualFold(it.Mod, want) || strings.EqualFold(it.Name, want) || strings.EqualFold(it.Key, want) {
		return true
	}
	if it.File != "" && (strings.EqualFold(it.Mod+"/"+it.File, want) || strings.EqualFold(it.File, want)) {
		return true
	}
	return false
}
