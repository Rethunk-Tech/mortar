package control

import (
	"context"
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/problems"
)

func (s *Services) dismissProblem(ctx context.Context, gameID, profileID string, index int) error {
	if index < 1 {
		return fmt.Errorf("invalid problem index %d", index)
	}
	res, err := s.Problems.Problems(ctx, gameID, profileID)
	if err != nil {
		return err
	}
	rows := problems.DismissableRows(res)
	if index > len(rows) {
		return fmt.Errorf("problem index %d out of range (%d dismissable)", index, len(rows))
	}
	row := rows[index-1]
	switch row.Kind {
	case "listed":
		return s.Problems.DismissListedRequirement(ctx, gameID, profileID, row.UniqueID)
	case "abandoned":
		return s.Problems.DismissAbandonedMod(ctx, gameID, profileID, row.UniqueID)
	case "conflict":
		return s.Problems.DismissAssetConflict(ctx, gameID, profileID, row.ConflictKind, row.Target)
	case "setting":
		return s.Problems.DismissSetting(ctx, gameID, profileID, row.UniqueID, row.Field)
	default:
		return fmt.Errorf("cannot dismiss problem at index %d", index)
	}
}

func (s *Services) restoreDismissedProblem(ctx context.Context, gameID, profileID, token string, index int) error {
	if token == "" {
		if index < 1 {
			return fmt.Errorf("missing dismissal token or index")
		}
		res, err := s.Problems.Problems(ctx, gameID, profileID)
		if err != nil {
			return err
		}
		if index > len(res.Dismissed) {
			return fmt.Errorf("dismissed index %d out of range (%d dismissed)", index, len(res.Dismissed))
		}
		token = res.Dismissed[index-1].Token
	}
	return s.Problems.RestoreDismissed(ctx, gameID, profileID, token)
}
