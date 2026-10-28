// Package scratch makes the temporary folders for large unpacks.
package scratch

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Dir makes a folder under the data folder's cache, where the player's disk has room and the cache cleanup reaches
// leftovers, not under the system temp folder, which may be a small tmpfs.
func Dir(pattern string) (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	cache := filepath.Join(base, "cache")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(cache, pattern)
}
