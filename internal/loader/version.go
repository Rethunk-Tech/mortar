package loader

import "strings"

// VersionChanged reports that the installed game version is known and differs from the version recorded at the last
// successful launch.
func VersionChanged(recorded, installed string) bool {
	recorded, installed = strings.TrimSpace(recorded), strings.TrimSpace(installed)
	if recorded == "" || installed == "" {
		return false
	}
	return recorded != installed
}
