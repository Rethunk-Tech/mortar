package dlwatch

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var nexusArchive = regexp.MustCompile(`(?i)^(.+?)-(\d+)-(\d+(?:-\d+)*)-(\d{9,})\.(zip|7z|rar)$`)

// Info is what Mortar knows about a noticed archive before install.
type Info struct {
	Name  string
	ModID int
}

// ParseNexusFilename reads the Nexus Mod Manager archive pattern
// "<name>-<modId>-<version>-<timestamp>.<ext>".
func ParseNexusFilename(name string) (Info, bool) {
	base := filepath.Base(name)
	m := nexusArchive.FindStringSubmatch(base)
	if m == nil {
		return Info{}, false
	}
	id, err := strconv.Atoi(m[2])
	if err != nil || id <= 0 {
		return Info{}, false
	}
	label := strings.ReplaceAll(m[1], "_", " ")
	return Info{Name: label, ModID: id}, true
}

func archiveExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".zip", ".7z", ".rar":
		return true
	default:
		return false
	}
}
