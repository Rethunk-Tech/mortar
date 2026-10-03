package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

// Item is one store folder in a cleanup report.
type Item struct {
	Key      string    `json:"key"`
	Name     string    `json:"name"`
	Version  string    `json:"version"`
	LastUsed time.Time `json:"lastUsed"`
	Size     int64     `json:"size"`
}

// GameReport is unused items and duplicate groups for one game.
type GameReport struct {
	Unused     []Item   `json:"unused"`
	Duplicates [][]Item `json:"duplicates"`
}

// Report is a cleanup view of the store, keyed by game id.
type Report map[string]GameReport

type tagged struct {
	item     Item
	uniqueID string
}

// Report lists items no profile names, and groups whose manifest UniqueID and Version match.
func (s *Store) Report(referenced map[string][]string) (Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.migrateCompleteMarkers(); err != nil {
		return nil, err
	}
	idx, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	games, err := os.ReadDir(s.root)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	out := Report{}
	for _, g := range games {
		if !g.IsDir() || !game.Valid(g.Name()) {
			continue
		}
		rep, err := s.gameReport(g.Name(), idx, keepSet(referenced[g.Name()]))
		if err != nil {
			return nil, err
		}
		if len(rep.Unused) == 0 && len(rep.Duplicates) == 0 {
			continue
		}
		out[g.Name()] = rep
	}
	return out, nil
}

func (s *Store) gameReport(gameID string, idx index, keep map[string]bool) (GameReport, error) {
	items, err := os.ReadDir(filepath.Join(s.root, gameID))
	if err != nil {
		return GameReport{}, err
	}
	var unused []Item
	var taggedItems []tagged
	for _, it := range items {
		key := it.Name()
		if !it.IsDir() || strings.HasPrefix(key, tempPrefix) || !keyPattern.MatchString(key) {
			continue
		}
		dir := filepath.Join(s.root, gameID, key)
		name, uniqueID, version := readManifest(dir)
		entry := Item{
			Key: key, Name: name, Version: version, LastUsed: idx[gameID][key], Size: dirSize(dir),
		}
		if !keep[key] {
			unused = append(unused, entry)
		}
		taggedItems = append(taggedItems, tagged{item: entry, uniqueID: uniqueID})
	}
	slices.SortFunc(unused, func(a, b Item) int { return strings.Compare(a.Key, b.Key) })
	return GameReport{Unused: unused, Duplicates: duplicateGroups(taggedItems)}, nil
}

func duplicateGroups(items []tagged) [][]Item {
	byID := map[string][]Item{}
	for _, it := range items {
		if it.uniqueID == "" || it.item.Version == "" {
			continue
		}
		id := it.uniqueID + "\x00" + it.item.Version
		if containsKey(byID[id], it.item.Key) {
			continue
		}
		byID[id] = append(byID[id], it.item)
	}
	var groups [][]Item
	for _, group := range byID {
		if len(group) >= 2 {
			slices.SortFunc(group, func(a, b Item) int { return strings.Compare(a.Key, b.Key) })
			groups = append(groups, group)
		}
	}
	slices.SortFunc(groups, func(a, b []Item) int { return strings.Compare(a[0].Key, b[0].Key) })
	return groups
}

func containsKey(items []Item, key string) bool {
	for _, it := range items {
		if it.Key == key {
			return true
		}
	}
	return false
}

func readManifest(dir string) (name, uniqueID, version string) {
	mods, err := manifest.Scan(dir)
	if err != nil || len(mods) == 0 {
		return "", "", ""
	}
	return mods[0].Name, mods[0].UniqueID, mods[0].Version
}
