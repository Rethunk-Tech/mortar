package stardew

import (
	"strings"
)

// StardewVersionFromLog returns the game version in a SMAPI log header
// ("SMAPI x with Stardew Valley y").
func StardewVersionFromLog(log string) string {
	for line := range strings.SplitSeq(log, "\n") {
		if m := logHeader.FindStringSubmatch(strings.TrimRight(line, "\r")); m != nil {
			return m[2]
		}
	}
	return ""
}

// GameVersionChanged reports that the installed game version is known and
// differs from the version recorded at the last successful launch.
func GameVersionChanged(recorded, installed string) bool {
	recorded = strings.TrimSpace(recorded)
	installed = strings.TrimSpace(installed)
	if recorded == "" || installed == "" {
		return false
	}
	return recorded != installed
}
