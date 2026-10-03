package profile

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const historyFilesDir = "history-files"

func captureHistoryConfigs(dir, snapshotID string, entries []Entry) {
	if snapshotID == "" {
		return
	}
	root := filepath.Join(dir, historyFilesDir, snapshotID)
	_ = os.RemoveAll(root)
	mods := filepath.Join(dir, "mods")
	for _, e := range entries {
		files := configFilesIn(liveEntryDir(mods, e.Key))
		if len(files) == 0 {
			continue
		}
		for rel, body := range files {
			path := filepath.Join(root, e.Key, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return
			}
			if err := datadir.WriteFile(path, body, 0o600); err != nil {
				return
			}
		}
	}
}

func loadHistoryConfigs(dir, snapshotID string, entries []Entry) map[string]map[string][]byte {
	out := emptyConfigs()
	if snapshotID == "" {
		return out
	}
	root := filepath.Join(dir, historyFilesDir, snapshotID)
	for _, e := range entries {
		id := entryIdentity(e)
		files := map[string][]byte{}
		base := filepath.Join(root, e.Key)
		_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(base, path)
			if relErr != nil {
				return nil
			}
			slash := filepath.ToSlash(rel)
			if !datadir.WritableRel(slash) {
				return nil
			}
			body, readErr := fsx.ReadFile(path)
			if readErr != nil {
				return nil
			}
			files[slash] = body
			return nil
		})
		if len(files) > 0 {
			out[id] = files
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
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if !datadir.WritableRel(slash) {
			return nil
		}
		body, readErr := fsx.ReadFile(path)
		if readErr != nil {
			return nil
		}
		files[slash] = body
		return nil
	})
	return files
}

func restoreHistoryConfig(dir, snapshotID, key, file string) error {
	src := filepath.Join(dir, historyFilesDir, snapshotID, key, filepath.FromSlash(file))
	body, err := fsx.ReadFile(src)
	if err != nil {
		return err
	}
	dst := filepath.Join(liveEntryDir(filepath.Join(dir, "mods"), key), filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return datadir.WriteFile(dst, body, 0o600)
}

func predecessorOf(data historyFileData, eventID string) (before []Entry, beforeID string, event HistoryEvent, ok bool) {
	for i, ev := range data.Events {
		if ev.ID != eventID {
			continue
		}
		if i > 0 {
			prev := data.Events[i-1]
			before, ok = snapshotEntries(data, prev.SnapshotID)
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
