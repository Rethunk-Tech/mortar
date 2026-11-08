package control

import (
	"context"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/profile"
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
	if s.Launches == nil {
		return s.Profiles.ChangesSince(game, id, "")
	}
	started, err := s.Launches.LastRunStarted(game, id)
	if err != nil {
		return profile.HistoryDiff{}, err
	}
	return s.Profiles.ChangesSince(game, id, started)
}

func (s *Services) profileGood(game, id string, mark, restore bool) (any, error) {
	switch {
	case mark && restore:
		return nil, fmt.Errorf("profile good takes --mark or --restore, not both")
	case mark:
		return s.changed(game, func() (any, error) { return s.Profiles.MarkKnownGood(game, id) })
	case restore:
		return s.changed(game, func() (any, error) { return s.Profiles.RestoreKnownGood(game, id) })
	default:
		return s.Profiles.KnownGood(game, id)
	}
}

func (s *Services) changesPlayGroup(ctx context.Context, game, id string) PlayIssueGroup {
	_ = ctx
	diff, err := s.profileChanges(game, id)
	if err != nil || len(diff.Items) == 0 {
		return PlayIssueGroup{}
	}
	names := make([]string, 0, min(len(diff.Items), playIssueNameCap))
	for _, it := range diff.Items {
		if it.Detail != "" {
			names = append(names, it.Detail)
		}
	}
	if len(names) > playIssueNameCap {
		names = names[:playIssueNameCap]
	}
	return PlayIssueGroup{Kind: "changes", Count: len(diff.Items), Names: names}
}
