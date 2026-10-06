package saves

import (
	"path"
	"regexp"
	"strings"
)

var lcSlot = regexp.MustCompile(`^LCSaveFile(\d+)$`)

// DisplayName is a save's name as the app shows it: its farm, else Lethal Company's slot (its saves are files named by
// slot), else its folder or file name without the extension (a Valheim world is worlds_local/<name>.fwl).
func DisplayName(farm, folder string) string {
	if farm != "" {
		return farm
	}
	if m := lcSlot.FindStringSubmatch(folder); m != nil {
		return "Save file " + m[1]
	}
	if folder == "LCChallengeFile" {
		return "Challenge moon"
	}
	base := path.Base(folder)
	return strings.TrimSuffix(base, path.Ext(base))
}
