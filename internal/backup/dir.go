package backup

import (
	"path/filepath"
	"strings"
)

// Locations returns the folder new backups are written to and the folders to
// read from. An empty custom path uses <dataDir>/backups; existing zips in the
// default folder stay listed after the write folder changes.
func Locations(dataDir, custom string) (write string, reads []string, err error) {
	base := dataDir
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
