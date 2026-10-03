package profile

import (
	"fmt"
	"time"
)

const historyGood = "good"

// MarkKnownGood records the current entries as a known-good restore point.
func (s *Store) MarkKnownGood(game, id string) (HistoryEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return HistoryEvent{}, err
	}
	p, err := s.read(game, id)
	if err != nil {
		return HistoryEvent{}, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return HistoryEvent{}, err
	}
	return appendHistory(dir, HistoryEvent{
		Kind: historyGood, Label: "Known good", Count: 1, At: time.Now().UTC(),
	}, p.Entries, s.historyKeep())
}

// RestoreKnownGood restores the newest known-good snapshot.
func (s *Store) RestoreKnownGood(game, id string) (Profile, error) {
	eventID, err := s.lastKnownGoodID(game, id)
	if err != nil {
		return Profile{}, err
	}
	return s.Revert(game, id, eventID)
}

// KnownGood lists known-good history events, newest first.
func (s *Store) KnownGood(game, id string) ([]HistoryEvent, error) {
	events, err := s.History(game, id)
	if err != nil {
		return nil, err
	}
	var out []HistoryEvent
	for _, ev := range events {
		if ev.Kind == historyGood {
			out = append(out, ev)
		}
	}
	if out == nil {
		out = []HistoryEvent{}
	}
	return out, nil
}

func (s *Store) lastKnownGoodID(game, id string) (string, error) {
	events, err := s.KnownGood(game, id)
	if err != nil {
		return "", err
	}
	if len(events) == 0 {
		return "", fmt.Errorf("no known-good snapshot")
	}
	return events[0].ID, nil
}

func titleRunOK(outcome string, errors int, crashed bool) bool {
	if crashed || errors > 0 {
		return false
	}
	switch outcome {
	case "ran", "ok", "success", "succeeded", "exited", "":
		return true
	default:
		return false
	}
}
