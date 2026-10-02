package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/problems"
)

// dismissRow is one problem row mortar problems may dismiss by index.
type dismissRow struct {
	kind         string
	uniqueID     string
	field        string
	conflictKind string
	target       string
}

func dismissableRows(r problems.Result) []dismissRow {
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

func dismissedKindText(d problems.DismissedProblem) (kind, text string) {
	if d.AssetConflict != nil {
		c := d.AssetConflict
		return "conflict", fmt.Sprintf("%s %s: %s", c.Kind, c.Target, strings.Join(c.Names, ", "))
	}
	if d.Broken != nil {
		b := d.Broken
		return "abandoned", fmt.Sprintf("%s: %s %s", b.Name, b.Status, b.Summary)
	}
	if d.Missing != nil {
		return "listed", fmt.Sprintf("%s needs %s", d.Missing.DependentName, missingName(*d.Missing))
	}
	if d.Setting != nil {
		s := d.Setting
		return "setting", fmt.Sprintf("%s %s=%s for %s", s.Name, s.Field, s.Current, strings.Join(s.ForNames, ", "))
	}
	return "unknown", d.Token
}

func parseProblemIndex(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 {
		return 0, usageError{fmt.Sprintf("invalid problem index %q", s)}
	}
	return n, nil
}

func (c *cmd) problemsGame() string {
	if c.game != "" {
		return c.game
	}
	return "stardew"
}

func (c *cmd) problemsProfile() (string, error) {
	if c.profileFlag == "" {
		return "", usageError{"problems needs --profile <name>"}
	}
	return c.profileFlag, nil
}
