package store

import (
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
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
	item Item
	id   mod.ID
}

// Report lists items no profile names, and groups whose manifest mod id and Version match.
func (s *Store) Report(referenced map[string][]string) (Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	out := Report{}
	for _, game := range slices.Sorted(maps.Keys(idx)) {
		rep := s.gameReport(idx[game], keepSet(referenced[game]))
		if len(rep.Unused) == 0 && len(rep.Duplicates) == 0 {
			continue
		}
		out[game] = rep
	}
	return out, nil
}

func (s *Store) gameReport(keys map[string]record, keep map[string]bool) GameReport {
	var unused []Item
	var taggedItems []tagged
	for key, r := range keys {
		dir, err := s.destOf(key, r.Blob)
		if err != nil || !completeItem(dir) {
			continue
		}
		name, id, version := readManifest(dir)
		size, _ := datadir.Size(dir)
		entry := Item{Key: key, Name: name, Version: version, LastUsed: r.Used, Size: size}
		if !keep[key] {
			unused = append(unused, entry)
		}
		taggedItems = append(taggedItems, tagged{item: entry, id: id})
	}
	slices.SortFunc(unused, func(a, b Item) int { return strings.Compare(a.Key, b.Key) })
	return GameReport{Unused: unused, Duplicates: duplicateGroups(taggedItems)}
}

func duplicateGroups(items []tagged) [][]Item {
	byID := map[string][]Item{}
	for _, it := range items {
		if it.id == "" || it.item.Version == "" {
			continue
		}
		id := string(it.id) + "\x00" + it.item.Version
		if slices.ContainsFunc(byID[id], func(x Item) bool { return x.Key == it.item.Key }) {
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

func readManifest(dir string) (name string, id mod.ID, version string) {
	mods, err := manifest.Scan(dir)
	if err != nil || len(mods) == 0 {
		return "", "", ""
	}
	return mods[0].Name, mods[0].ModID(), mods[0].Version
}
