package backup

import (
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Locations returns the folder new backups of game are written to and the folders to read from. Every location holds
// one folder per game, so one game's backups are never listed or restored for another. An empty custom path uses
// <dataDir>/backups; existing zips in the default folder stay listed after the write folder changes.
func Locations(dataDir, custom, game string) (write string, reads []string, err error) {
	def := filepath.Join(dataDir, "backups", game)
	custom = strings.TrimSpace(custom)
	if custom == "" {
		return def, []string{def}, nil
	}
	custom = filepath.Join(custom, game)
	reads = []string{custom}
	if !fsx.SamePath(custom, def) {
		reads = append(reads, def)
	}
	return custom, reads, nil
}
