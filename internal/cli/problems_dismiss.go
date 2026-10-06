package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/problems"
)

// problems.Dismissable is one problem row mortar problems may dismiss by index.
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
		return "setting", settingText(*d.Setting)
	}
	return "unknown", d.Token
}

// settingText is a setting row: the mods it is for, or, for a setting with no such mods, why it is listed.
func settingText(s framework.SettingHint) string {
	if len(s.ForNames) == 0 {
		return fmt.Sprintf("%s %s=%s: %s", s.Name, s.Field, s.Current, s.Description)
	}
	return fmt.Sprintf("%s %s=%s for %s", s.Name, s.Field, s.Current, strings.Join(s.ForNames, ", "))
}

func parseProblemIndex(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 {
		return 0, usageError{fmt.Sprintf("invalid problem index %q", s)}
	}
	return n, nil
}

func (c *cmd) problemsProfile() (string, error) {
	if c.profileFlag == "" {
		return "", usageError{"problems needs --profile <name>"}
	}
	return c.profileFlag, nil
}
