package backup

import (
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Locations returns the folder new backups are written to and the folders to
// read from. An empty custom path uses <datadir>/backups; existing zips in the
// default folder stay listed after the write folder changes.
func Locations(custom string) (write string, reads []string, err error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", nil, err
	}
	def := filepath.Join(base, "backups")
	custom = strings.TrimSpace(custom)
	if custom == "" {
		return def, []string{def}, nil
	}
	reads = []string{custom}
	if filepath.Clean(custom) != filepath.Clean(def) {
		reads = append(reads, def)
	}
	return custom, reads, nil
}
