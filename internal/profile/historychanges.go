package profile

import (
	"slices"
	"time"
)

// ChangesSince returns the diff from the newest snapshot at or before since to the latest snapshot.
func (s *Store) ChangesSince(game, id string, since time.Time) (HistoryDiff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.read(game, id); err != nil {
		return HistoryDiff{}, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return HistoryDiff{}, err
	}
	data, err := readHistory(dir)
	if err != nil {
		return HistoryDiff{}, err
	}
	if since.IsZero() || len(data.Events) == 0 {
		return HistoryDiff{}, nil
	}
	a, b := snapshotsAround(data, since)
	if a == "" && b == "" {
		return HistoryDiff{}, nil
	}
	var before []Entry
	if a != "" {
		before, _ = snapshotEntries(data, a)
	}
	after, ok := snapshotEntries(data, b)
	if !ok {
		return HistoryDiff{}, nil
	}
	cfgA := loadHistoryConfigs(dir, snapshotHash(data, a), before)
	cfgB := loadHistoryConfigs(dir, snapshotHash(data, b), after)
	overlayLiveIfCurrent(dir, data, after, cfgB)
	return DiffSnapshots(a, b, before, after, cfgA, cfgB), nil
}

func snapshotsAround(data historyFileData, since time.Time) (a, b string) {
	if len(data.Events) == 0 {
		return "", ""
	}
	last := data.Events[len(data.Events)-1]
	b = last.ID
	for _, ev := range slices.Backward(data.Events) {
		if !ev.At.After(since) {
			return ev.ID, b
		}
	}
	return "", b
}

// EventDiff is the change set of one history event against its predecessor.
func (s *Store) EventDiff(game, id, eventID string) (HistoryDiff, error) {
	_, _, diff, err := s.eventDiff(game, id, eventID)
	return diff, err
}
