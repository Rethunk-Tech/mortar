package profile

import (
	"fmt"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/store"
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

//wails:ignore
func (s *Service) KnownGood(game, id string) ([]HistoryEvent, error) {
	return s.store.KnownGood(game, id)
}

func (s *Service) HistoryUsage(game string) ([]HistoryUsage, error) {
	return s.store.HistoryUsage(game)
}

func (s *Service) TrimHistory(game, id string, keepLast int) (HistoryUsage, error) {
	return s.store.TrimHistory(game, id, keepLast)
}

// AllowUnscanned is the player's "Install anyway" for an item the antivirus flagged: the next install of key skips the
// scan, and once that install has succeeded the profile's history records that it was allowed, naming the mod and what
// was flagged.
func (s *Service) AllowUnscanned(game, id, key, name, detection string) error {
	return s.store.AllowUnscanned(game, id, key, name, detection)
}

// AllowUnscanned makes the store skip the antivirus scan once for key; installKey records the choice once the item is in.
func (s *Store) AllowUnscanned(game, id, key, name, detection string) error {
	s.items.AllowUnscanned(game, key, store.Override{Profile: id, Name: name, Detection: detection})
	return nil
}

// recordOverride writes the history entry for an "Install anyway" whose install has just succeeded.
func (s *Store) recordOverride(game, id, key string) error {
	ov, ok := s.items.TakeOverride(game, key)
	if !ok || ov.Profile != id {
		return nil
	}
	_, err := s.recordSnapshot(game, id, historyBulk, HistoryEvent{Change: ChangeUnscanned, Name: ov.Name, Detail: ov.Detection}, 1)
	return err
}
