package control

import (
	"context"
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/problems"
)

type dismissRow struct {
	kind         string
	uniqueID     string
	field        string
	conflictKind string
	target       string
}

func dismissableProblems(r problems.Result) []dismissRow {
	var out []dismissRow
	for _, x := range r.Missing {
		if x.Listed {
			out = append(out, dismissRow{kind: "listed", uniqueID: x.UniqueID})
		}
	}
	for _, x := range r.Broken {
		if x.Status == "abandoned" {
			out = append(out, dismissRow{kind: "abandoned", uniqueID: x.UniqueID})
		}
	}
	for _, x := range r.AssetConflicts {
		if !x.Cosmetic && x.Kind != "" {
			out = append(out, dismissRow{kind: "conflict", conflictKind: x.Kind, target: x.Target})
		}
	}
	for _, x := range r.Settings {
		out = append(out, dismissRow{kind: "setting", uniqueID: x.UniqueID, field: x.Field})
	}
	return out
}

func (s *Services) dismissProblem(ctx context.Context, gameID, profileID string, index int) error {
	if index < 1 {
		return fmt.Errorf("invalid problem index %d", index)
	}
	res, err := s.Problems.Problems(ctx, gameID, profileID)
	if err != nil {
		return err
	}
	rows := dismissableProblems(res)
	if index > len(rows) {
		return fmt.Errorf("problem index %d out of range (%d dismissable)", index, len(rows))
	}
	row := rows[index-1]
	switch row.kind {
	case "listed":
		return s.Problems.DismissListedRequirement(ctx, gameID, profileID, row.uniqueID)
	case "abandoned":
		return s.Problems.DismissAbandonedMod(ctx, gameID, profileID, row.uniqueID)
	case "conflict":
		return s.Problems.DismissAssetConflict(ctx, gameID, profileID, row.conflictKind, row.target)
	case "setting":
		return s.Problems.DismissSetting(ctx, gameID, profileID, row.uniqueID, row.field)
	default:
		return fmt.Errorf("cannot dismiss problem at index %d", index)
	}
}

func (s *Services) restoreDismissedProblem(ctx context.Context, gameID, profileID string, token string, index int) error {
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
