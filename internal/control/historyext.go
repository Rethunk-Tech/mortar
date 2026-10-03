package control

import (
	"context"
	"fmt"
	"time"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (s *Services) historyDiff(game, id, a, b string) (profile.HistoryDiff, error) {
	return s.Profiles.HistoryDiff(game, id, a, b)
}

func (s *Services) historyRevertItem(game, id, eventID, item string) (any, error) {
	if item == "" {
		return nil, fmt.Errorf("history revert needs --item")
	}
	return s.changed(game, func() (any, error) {
		return s.Profiles.RevertHistoryItem(game, id, eventID, item)
	})
}

func (s *Services) profileChanges(game, id string) (profile.HistoryDiff, error) {
	since, err := s.lastRunTime(game, id)
	if err != nil {
		return profile.HistoryDiff{}, err
	}
	if since.IsZero() {
		return s.Profiles.ChangesSince(game, id, "")
	}
	return s.Profiles.ChangesSince(game, id, since.Format(time.RFC3339Nano))
}

func (s *Services) lastRunTime(game, id string) (time.Time, error) {
	if s.Launches == nil {
		return time.Time{}, nil
	}
	runs, err := s.Launches.Runs(game, id)
	if err != nil || len(runs) == 0 {
		return time.Time{}, err
	}
	return parseSinceTime(runs[0].Started), nil
}

func parseSinceTime(raw string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (s *Services) profileGood(game, id string, mark, restore bool) (any, error) {
	switch {
	case mark && restore:
		return nil, fmt.Errorf("profile good needs --mark or --restore")
	case mark:
		return s.changed(game, func() (any, error) { return s.Profiles.MarkKnownGood(game, id) })
	case restore:
		return s.changed(game, func() (any, error) { return s.Profiles.RestoreKnownGood(game, id) })
	default:
		return s.Profiles.KnownGood(game, id)
	}
}

const playChangeNames = 8

func (s *Services) changesPlayGroup(ctx context.Context, game, id string) PlayIssueGroup {
	_ = ctx
	diff, err := s.profileChanges(game, id)
	if err != nil || len(diff.Items) == 0 {
		return PlayIssueGroup{}
	}
	names := make([]string, 0, len(diff.Items))
	for _, it := range diff.Items {
		if it.Detail != "" {
			names = append(names, it.Detail)
		}
	}
	if len(names) > playChangeNames {
		names = names[:playChangeNames]
	}
	return PlayIssueGroup{Kind: "changes", Count: len(diff.Items), Names: names}
}
