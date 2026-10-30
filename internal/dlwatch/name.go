package dlwatch

import (
	"strings"

	"github.com/Rethunk-Tech/go-nexusmods"
)

// Info is what Mortar knows about a noticed archive before install.
type Info struct {
	Name  string
	ModID int
}

// ParseNexusFilename reads the Nexus Mod Manager archive pattern
// "<name>-<modId>-<version>-<timestamp>.<ext>", with the underscores Nexus puts in a name shown as spaces.
func ParseNexusFilename(name string) (Info, bool) {
	a, ok := nexusmods.ParseArchiveName(name)
	return Info{Name: strings.ReplaceAll(a.Name, "_", " "), ModID: a.ModID}, ok
}
