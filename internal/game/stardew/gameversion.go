package stardew

import "strings"

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
