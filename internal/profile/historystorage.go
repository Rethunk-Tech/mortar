package profile

import (
	"fmt"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
)

const historyTrimmed = "trimmed"

// HistoryUsage is what one profile's change history takes on disk.
type HistoryUsage struct {
	ProfileID     string `json:"profileId"`
	ProfileName   string `json:"profileName"`
	Events        int    `json:"events"`
	SnapshotBytes int64  `json:"snapshotBytes"`
	FileBytes     int64  `json:"fileBytes"`
}

func historyUsageAt(dir string, p Profile, events int) HistoryUsage {
	snaps, _ := safefs.Size(filepath.Join(dir, snapshotsDir))
	files, _ := safefs.Size(filepath.Join(dir, historyFilesDir))
	return HistoryUsage{ProfileID: p.ID, ProfileName: p.Name, Events: events, SnapshotBytes: snaps, FileBytes: files}
}

// HistoryUsage sizes the history of each usable profile of the game: its snapshots and its kept config files.
func (s *Store) HistoryUsage(gameID string) ([]HistoryUsage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.listOK(gameID)
	if err != nil {
		return nil, err
	}
	out := make([]HistoryUsage, 0, len(list))
	for _, p := range list {
		dir, err := s.profileDir(gameID, p.ID)
		if err != nil {
			return nil, err
		}
		data, err := readHistory(dir)
		if err != nil {
			return nil, err
		}
		out = append(out, historyUsageAt(dir, p, len(data.Events)))
	}
	return out, nil
}

// TrimHistory drops all but the newest keepLast events of a profile with the snapshots and files only they used,
// then records one event saying so.
func (s *Store) TrimHistory(gameID, id string, keepLast int) (HistoryUsage, error) {
	if keepLast < 1 {
		return HistoryUsage{}, fmt.Errorf("keep at least one history event")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(gameID, id)
	if err != nil {
		return HistoryUsage{}, err
	}
	data, err := readHistory(dir)
	if err != nil {
		return HistoryUsage{}, err
	}
	dropped := len(data.Events) - keepLast
	if dropped > 0 {
		if err := writeHistory(dir, data, keepLast); err != nil {
			return HistoryUsage{}, err
		}
		if _, err := appendHistory(dir, HistoryEvent{Kind: historyTrimmed, Change: ChangeTrimmed, Count: dropped}, p.Entries, keepLast+1); err != nil {
			return HistoryUsage{}, err
		}
		data, err = readHistory(dir)
		if err != nil {
			return HistoryUsage{}, err
		}
	}
	return historyUsageAt(dir, p, len(data.Events)), nil
}
