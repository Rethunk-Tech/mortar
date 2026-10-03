package profile

import (
	"fmt"
	"strings"
	"time"
)

func parseSince(since string) (time.Time, error) {
	since = strings.TrimSpace(since)
	if since == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04"} {
		if t, err := time.Parse(layout, since); err == nil {
			return t, nil
		}
		if t, err := time.ParseInLocation(layout, since, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q", since)
}

func (s *Service) HistoryDiff(game, id, a, b string) (HistoryDiff, error) {
	return s.store.HistoryDiff(game, id, a, b)
}

func (s *Service) EventDiff(game, id, eventID string) (HistoryDiff, error) {
	return s.store.EventDiff(game, id, eventID)
}

func (s *Service) RevertHistoryItem(game, id, eventID, item string) (Profile, error) {
	return s.store.RevertHistoryItem(game, id, eventID, item)
}

func (s *Service) ChangesSince(game, id, since string) (HistoryDiff, error) {
	when, err := parseSince(since)
	if err != nil {
		return HistoryDiff{}, err
	}
	return s.store.ChangesSince(game, id, when)
}

func (s *Service) MarkKnownGood(game, id string) (HistoryEvent, error) {
	return s.store.MarkKnownGood(game, id)
}

func (s *Service) RestoreKnownGood(game, id string) (Profile, error) {
	return s.store.RestoreKnownGood(game, id)
}

func (s *Service) KnownGood(game, id string) ([]HistoryEvent, error) {
	return s.store.KnownGood(game, id)
}
