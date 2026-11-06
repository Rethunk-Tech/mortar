// Package scratch makes the temporary folders for large unpacks.
package scratch

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Dir makes a folder under the data folder's cache, where the player's disk has room and the cache cleanup reaches
// leftovers, not under the system temp folder, which may be a small tmpfs.
func Dir(pattern string) (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return DirIn(base, pattern)
}

// DirIn makes the folder under the cache of the data folder base, for a caller that holds its own data folder.
func DirIn(base, pattern string) (string, error) {
	cache := filepath.Join(base, "cache")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(cache, pattern)
}

// Sweep removes the scratch folders under cache that nothing has touched for maxAge, the leftovers of a run that died
// mid-unpack. It reports how many it removed.
func Sweep(maxAge time.Duration) (int, error) {
	base, err := datadir.Dir()
	if err != nil {
		return 0, err
	}
	dirs, err := filepath.Glob(filepath.Join(base, "cache", "mortar-*"))
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, d := range dirs {
		info, err := os.Lstat(d)
		if err != nil || !info.IsDir() || time.Since(info.ModTime()) < maxAge {
			continue
		}
		if err := safefs.RemoveAll(d); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
