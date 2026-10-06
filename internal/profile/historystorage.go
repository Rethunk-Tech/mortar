package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
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
	snaps, _ := datadir.Size(filepath.Join(dir, snapshotsDir))
	files, _ := datadir.Size(filepath.Join(dir, historyFilesDir))
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

// MigrateHistory gzips every remaining plain snapshot and fills in missing event counts for all live profiles, so
// space is reclaimed without opening each history. It returns how many snapshots and histories it converted.
func (s *Store) MigrateHistory() (snapshots, counted int, err error) {
	gameDirs, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	for _, g := range gameDirs {
		if !g.IsDir() || !game.Valid(g.Name()) {
			continue
		}
		profiles, err := os.ReadDir(filepath.Join(s.root, g.Name()))
		if err != nil {
			return snapshots, counted, err
		}
		for _, d := range profiles {
			if !d.IsDir() || !idPattern.MatchString(d.Name()) {
				continue
			}
			n, c, err := s.migrateHistoryOf(filepath.Join(s.root, g.Name(), d.Name()))
			if err != nil {
				return snapshots, counted, err
			}
			snapshots += n
			if c {
				counted++
			}
		}
	}
	return snapshots, counted, nil
}

func (s *Store) migrateHistoryOf(dir string) (snapshots int, counted bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(filepath.Join(dir, fileName)); err != nil {
		return 0, false, nil
	}
	data, err := readHistory(dir)
	if err != nil {
		return 0, false, err
	}
	ents, _ := os.ReadDir(filepath.Join(dir, snapshotsDir))
	for _, ent := range ents {
		name, plain := strings.CutSuffix(ent.Name(), ".json")
		if !plain || ent.IsDir() {
			continue
		}
		if _, ok := migratePlainSnapshot(dir, name, filepath.Join(dir, snapshotsDir, ent.Name())); ok {
			snapshots++
		}
	}
	if !data.Counted {
		countAllEvents(&data)
		if err := writeHistory(dir, data, 0); err != nil {
			return snapshots, false, err
		}
		counted = true
	}
	return snapshots, counted, nil
}
