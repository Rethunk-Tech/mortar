package profile

import (
	"fmt"
)

// RevertHistoryItem undoes one item of eventID through a normal profile mutation.
func (s *Store) RevertHistoryItem(game, id, eventID, item string) (Profile, error) {
	before, beforeSnap, diff, err := s.eventDiff(game, id, eventID)
	if err != nil {
		return Profile{}, err
	}
	hit, err := pickHistoryItem(diff.Items, item)
	if err != nil {
		return Profile{}, err
	}
	switch hit.Kind {
	case diffUpdated:
		return s.revertItemUpdate(game, id, hit)
	case diffEnabled, diffDisabled:
		return s.SetModEnabled(game, id, hit.Key, hit.Mod, hit.Kind == diffDisabled)
	case diffRemoved:
		return s.RestoreEntries(game, id, entriesFor(before, hit))
	case diffAdded:
		return s.RemoveEntry(game, id, hit.Key)
	case diffConfig:
		return s.revertItemConfig(game, id, beforeSnap, hit)
	default:
		return Profile{}, fmt.Errorf("cannot revert %s", hit.Kind)
	}
}

func (s *Store) eventDiff(game, id, eventID string) ([]Entry, string, HistoryDiff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadHistory(game, id)
	if err != nil {
		return nil, "", HistoryDiff{}, err
	}
	before, beforeID, ev, ok := predecessorOf(data, eventID)
	if !ok {
		return nil, "", HistoryDiff{}, fmt.Errorf("history event %s not found", eventID)
	}
	after, ok := snapshotEntries(&data, ev.SnapshotID)
	if !ok {
		return nil, "", HistoryDiff{}, fmt.Errorf("history snapshot %s not found", ev.SnapshotID)
	}
	cfgA := loadHistoryConfigs(data.dir, beforeID, before)
	cfgB := loadHistoryConfigs(data.dir, ev.SnapshotID, after)
	return before, beforeID, DiffSnapshots(beforeID, ev.ID, before, after, cfgA, cfgB), nil
}

func pickHistoryItem(items []HistoryItem, want string) (HistoryItem, error) {
	var hits []HistoryItem
	for _, it := range items {
		if itemMatches(it, want) {
			hits = append(hits, it)
		}
	}
	if len(hits) == 1 {
		return hits[0], nil
	}
	if len(hits) == 0 {
		return HistoryItem{}, fmt.Errorf("history item %s not found", want)
	}
	for _, kind := range []string{diffDisabled, diffEnabled, diffUpdated, diffRemoved, diffAdded, diffConfig} {
		var one []HistoryItem
		for _, hit := range hits {
			if hit.Kind == kind {
				one = append(one, hit)
			}
		}
		if len(one) == 1 {
			return one[0], nil
		}
	}
	return HistoryItem{}, fmt.Errorf("history item %s matches %d changes", want, len(hits))
}

func entriesFor(before []Entry, hit HistoryItem) []Entry {
	for _, e := range before {
		if entryIdentity(e) == hit.Mod || e.Key == hit.Key {
			return append([]Entry{e}, overlaysOf(before, e.Key)...)
		}
	}
	return nil
}

func (s *Store) revertItemUpdate(game, id string, hit HistoryItem) (Profile, error) {
	if hit.OldKey == "" {
		return Profile{}, fmt.Errorf("history item %s has no previous version", hit.Mod)
	}
	p, err := s.read(game, id)
	if err != nil {
		return Profile{}, err
	}
	cur := hit.NewKey
	if cur == "" {
		cur = hit.Key
	}
	for _, e := range p.Entries {
		if e.Key != cur {
			continue
		}
		if e.PreviousKey == hit.OldKey {
			return s.RollBack(game, id, e.Key)
		}
		return s.UpdateEntry(game, id, e.Key, hit.OldKey)
	}
	return Profile{}, fmt.Errorf("%s is not in this profile", hit.Name)
}

func (s *Store) revertItemConfig(game, id, beforeSnap string, hit HistoryItem) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	if err := restoreHistoryConfig(dir, beforeSnap, hit.Key, hit.File); err != nil {
		return Profile{}, err
	}
	p, err := s.read(game, id)
	if err != nil {
		return Profile{}, err
	}
	_, err = appendHistory(dir, HistoryEvent{Kind: historyReverted, Label: "Restored " + hit.Name + " " + hit.File, Count: 1}, p.Entries, s.historyKeep())
	if err != nil {
		return Profile{}, err
	}
	return p, nil
}
