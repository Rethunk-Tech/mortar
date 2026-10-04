package profile

import (
	"os"
	"path/filepath"
	"slices"
	"time"
)

type changesSinceCache struct {
	updated time.Time
	stamp   time.Time
	since   time.Time
	diff    HistoryDiff
}

// ChangesSince returns the diff from the newest snapshot at or before since to the latest snapshot.
func (s *Store) ChangesSince(game, id string, since time.Time) (HistoryDiff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return HistoryDiff{}, err
	}
	key := historyBatchKey(game, id)
	stamp := historyStamp(dir)
	if cached, ok := s.changesCache[key]; ok && cached.updated.Equal(p.Updated) && cached.stamp.Equal(stamp) && cached.since.Equal(since) {
		return cached.diff, nil
	}
	data, err := readHistory(dir)
	if err != nil {
		return HistoryDiff{}, err
	}
	stamp = historyStamp(dir)
	if since.IsZero() || len(data.Events) == 0 {
		s.storeChangesSince(key, p.Updated, stamp, since, HistoryDiff{})
		return HistoryDiff{}, nil
	}
	a, b := snapshotsAround(data, since)
	if a == "" && b == "" {
		s.storeChangesSince(key, p.Updated, stamp, since, HistoryDiff{})
		return HistoryDiff{}, nil
	}
	var before []Entry
	if a != "" {
		before, _ = snapshotEntries(&data, a)
	}
	after, ok := snapshotEntries(&data, b)
	if !ok {
		s.storeChangesSince(key, p.Updated, stamp, since, HistoryDiff{})
		return HistoryDiff{}, nil
	}
	cfgA := loadHistoryConfigs(dir, snapshotHash(data, a), before)
	cfgB := loadHistoryConfigs(dir, snapshotHash(data, b), after)
	overlayLiveIfCurrent(dir, data, after, cfgB)
	diff := DiffSnapshots(a, b, before, after, cfgA, cfgB)
	s.storeChangesSince(key, p.Updated, stamp, since, diff)
	return diff, nil
}

func historyStamp(dir string) time.Time {
	info, err := os.Stat(filepath.Join(dir, historyFile))
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func (s *Store) storeChangesSince(key string, updated, stamp, since time.Time, diff HistoryDiff) {
	if s.changesCache == nil {
		s.changesCache = map[string]changesSinceCache{}
	}
	s.changesCache[key] = changesSinceCache{updated: updated, stamp: stamp, since: since, diff: diff}
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
