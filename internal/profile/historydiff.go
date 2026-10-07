package profile

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

const (
	diffAdded    = "added"
	diffRemoved  = "removed"
	diffUpdated  = "updated"
	diffEnabled  = "enabled"
	diffDisabled = "disabled"
	diffConfig   = "config"
)

// HistoryDiff is the change set between two history snapshots.
type HistoryDiff struct {
	A        string        `json:"a"`
	B        string        `json:"b"`
	Added    []DiffMod     `json:"added"`
	Removed  []DiffMod     `json:"removed"`
	Versions []DiffVersion `json:"versions"`
	Enabled  []DiffEnabled `json:"enabled"`
	Configs  []DiffConfig  `json:"configs"`
	Items    []HistoryItem `json:"items"`
}

// DiffMod is a mod present on only one side of a snapshot pair.
type DiffMod struct {
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Key     string `json:"key"`
}

// DiffVersion is one mod's store key or version change.
type DiffVersion struct {
	ID     mod.ID `json:"id"`
	Name   string `json:"name"`
	Old    string `json:"old"`
	New    string `json:"new"`
	OldKey string `json:"oldKey"`
	NewKey string `json:"newKey"`
}

// DiffEnabled is one mod's enabled-state change.
type DiffEnabled struct {
	ID   mod.ID `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
	Old  bool   `json:"old"`
	New  bool   `json:"new"`
}

// DiffConfig lists config files that differ for one mod.
type DiffConfig struct {
	ID    mod.ID   `json:"id"`
	Name  string   `json:"name"`
	Key   string   `json:"key"`
	Files []string `json:"files"`
}

// HistoryItem is one revertible row in a snapshot pair.
type HistoryItem struct {
	Kind   string `json:"kind"`
	Mod    mod.ID `json:"mod"`
	Name   string `json:"name"`
	Key    string `json:"key"`
	OldKey string `json:"oldKey,omitempty"`
	NewKey string `json:"newKey,omitempty"`
	Old    string `json:"old,omitempty"`
	New    string `json:"new,omitempty"`
	File   string `json:"file,omitempty"`
	Detail string `json:"detail"`
}

// HistoryDiff compares snapshots a and b (event IDs or snapshot hashes).
func (s *Store) HistoryDiff(game, id, a, b string) (HistoryDiff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadHistory(game, id)
	if err != nil {
		return HistoryDiff{}, err
	}
	before, ok := snapshotEntries(&data, a)
	if !ok {
		return HistoryDiff{}, fmt.Errorf("history snapshot %s not found", a)
	}
	after, ok := snapshotEntries(&data, b)
	if !ok {
		return HistoryDiff{}, fmt.Errorf("history snapshot %s not found", b)
	}
	cfgA := loadHistoryConfigs(data.dir, snapshotHash(data, a), before)
	cfgB := loadHistoryConfigs(data.dir, snapshotHash(data, b), after)
	overlayLiveIfCurrent(data.dir, data, after, cfgB)
	return DiffSnapshots(a, b, before, after, cfgA, cfgB), nil
}

func snapshotHash(data historyFileData, id string) string {
	if _, ok := data.Snapshots[id]; ok {
		return id
	}
	for _, event := range data.Events {
		if event.ID == id {
			return event.SnapshotID
		}
	}
	return id
}

// DiffSnapshots compares two entry lists and optional config maps keyed by entry identity.
func DiffSnapshots(a, b string, before, after []Entry, cfgA, cfgB map[string]map[string][]byte) HistoryDiff {
	out := HistoryDiff{A: a, B: b}
	bMap := indexEntries(before)
	aMap := indexEntries(after)
	seen := map[string]struct{}{}
	for id, ae := range aMap {
		seen[id] = struct{}{}
		be, ok := bMap[id]
		if !ok {
			out.Added = append(out.Added, diffMod(ae))
			continue
		}
		appendEntryPair(&out, uniqueIDOf(ae, id), be, ae, cfgA[id], cfgB[id])
	}
	for id, be := range bMap {
		if _, ok := seen[id]; ok {
			continue
		}
		out.Removed = append(out.Removed, diffMod(be))
	}
	slices.SortFunc(out.Added, cmpDiffMod)
	slices.SortFunc(out.Removed, cmpDiffMod)
	slices.SortFunc(out.Versions, cmpDiffVersion)
	slices.SortFunc(out.Enabled, cmpDiffEnabled)
	slices.SortFunc(out.Configs, cmpDiffConfig)
	out.Items = historyItems(out)
	return out
}

func appendEntryPair(out *HistoryDiff, id mod.ID, be, ae Entry, oldCfg, newCfg map[string][]byte) {
	if be.Key != ae.Key || entryVersion(be) != entryVersion(ae) {
		out.Versions = append(out.Versions, DiffVersion{
			ID: id, Name: entryName(ae), Old: entryVersion(be), New: entryVersion(ae),
			OldKey: be.Key, NewKey: ae.Key,
		})
	}
	out.Enabled = append(out.Enabled, enabledChanges(be, ae)...)
	files := configFileDiff(oldCfg, newCfg)
	if len(files) > 0 {
		out.Configs = append(out.Configs, DiffConfig{ID: id, Name: entryName(ae), Key: ae.Key, Files: files})
	}
}

func uniqueIDOf(e Entry, fallback string) mod.ID {
	if len(e.Mods) > 0 && e.Mods[0].ID != "" {
		return e.Mods[0].ID
	}
	return mod.ID(fallback)
}

func diffMod(e Entry) DiffMod {
	return DiffMod{ID: uniqueIDOf(e, entryIdentity(e)), Name: entryName(e), Version: entryVersion(e), Key: e.Key}
}

func enabledChanges(before, after Entry) []DiffEnabled {
	if after.IsOverlay() {
		if before.OverlayOff == after.OverlayOff {
			return nil
		}
		return []DiffEnabled{{ID: mod.ID(after.Key), Name: entryName(after), Key: after.Key, Old: !before.OverlayOff, New: !after.OverlayOff}}
	}
	ids := uniqueIDs(before)
	for _, id := range uniqueIDs(after) {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	var out []DiffEnabled
	for _, id := range ids {
		oldOn := before.Enabled(id)
		newOn := after.Enabled(id)
		if oldOn == newOn {
			continue
		}
		out = append(out, DiffEnabled{ID: id, Name: modName(after, before, id), Key: after.Key, Old: oldOn, New: newOn})
	}
	return out
}

func uniqueIDs(e Entry) []mod.ID {
	if len(e.Mods) == 0 {
		return nil
	}
	out := make([]mod.ID, 0, len(e.Mods))
	for _, m := range e.Mods {
		out = append(out, m.ID)
	}
	return out
}

func modName(after, before Entry, uniqueID mod.ID) string {
	for _, e := range []Entry{after, before} {
		for _, m := range e.Mods {
			if m.ID == uniqueID {
				if m.Name != "" {
					return m.Name
				}
				return uniqueID.Local()
			}
		}
	}
	return uniqueID.Local()
}

func configFileDiff(oldCfg, newCfg map[string][]byte) []string {
	names := map[string]struct{}{}
	for name := range oldCfg {
		names[name] = struct{}{}
	}
	for name := range newCfg {
		names[name] = struct{}{}
	}
	var files []string
	for name := range names {
		if string(oldCfg[name]) != string(newCfg[name]) {
			files = append(files, name)
		}
	}
	slices.Sort(files)
	return files
}

func historyItems(d HistoryDiff) []HistoryItem {
	var items []HistoryItem
	for _, m := range d.Added {
		items = append(items, HistoryItem{Kind: diffAdded, Mod: m.ID, Name: m.Name, Key: m.Key, Detail: "Added " + m.Name})
	}
	for _, m := range d.Removed {
		items = append(items, HistoryItem{Kind: diffRemoved, Mod: m.ID, Name: m.Name, Key: m.Key, Detail: "Removed " + m.Name})
	}
	for _, v := range d.Versions {
		items = append(items, HistoryItem{
			Kind: diffUpdated, Mod: v.ID, Name: v.Name, Key: v.NewKey, OldKey: v.OldKey, NewKey: v.NewKey,
			Old: v.Old, New: v.New, Detail: fmt.Sprintf("%s %s → %s", v.Name, v.Old, v.New),
		})
	}
	for _, e := range d.Enabled {
		kind := diffEnabled
		if !e.New {
			kind = diffDisabled
		}
		items = append(items, HistoryItem{
			Kind: kind, Mod: e.ID, Name: e.Name, Key: e.Key, Old: enabledWord(e.Old), New: enabledWord(e.New),
			Detail: e.Name + " " + enabledWord(e.Old) + " → " + enabledWord(e.New),
		})
	}
	for _, c := range d.Configs {
		for _, file := range c.Files {
			items = append(items, HistoryItem{
				Kind: diffConfig, Mod: c.ID, Name: c.Name, Key: c.Key, File: file,
				Detail: c.Name + " " + file,
			})
		}
	}
	return items
}

func enabledWord(on bool) string {
	if on {
		return "enabled"
	}
	return "disabled"
}

func cmpDiffMod(a, b DiffMod) int { return strings.Compare(string(a.ID), string(b.ID)) }

func cmpDiffVersion(a, b DiffVersion) int { return strings.Compare(string(a.ID), string(b.ID)) }

func cmpDiffEnabled(a, b DiffEnabled) int { return strings.Compare(string(a.ID), string(b.ID)) }

func cmpDiffConfig(a, b DiffConfig) int { return strings.Compare(string(a.ID), string(b.ID)) }

func emptyConfigs() map[string]map[string][]byte {
	return map[string]map[string][]byte{}
}

func overlayLiveIfCurrent(dir string, data historyFileData, entries []Entry, cfg map[string]map[string][]byte) {
	latest, ok := latestSnapshotOf(data)
	if !ok || !entriesEqual(latest, entries) {
		return
	}
	overlayLiveConfigs(dir, entries, cfg)
}

func latestSnapshotOf(data historyFileData) ([]Entry, bool) {
	for _, ev := range slices.Backward(data.Events) {
		if entries, ok := snapshotEntries(&data, ev.SnapshotID); ok {
			return cloneEntries(entries), true
		}
	}
	return nil, false
}

func overlayLiveConfigs(dir string, entries []Entry, cfg map[string]map[string][]byte) {
	live := readLiveConfigs(filepath.Join(dir, "mods"), entries)
	for id, files := range live {
		if len(cfg[id]) == 0 {
			cfg[id] = files
		}
	}
}
