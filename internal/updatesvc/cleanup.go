package updatesvc

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// RemoveOldExecutables deletes the <exe>.old.<n> copies a Windows self-update leaves beside the running exe, since
// the old file cannot be removed while it runs. Best effort: one still locked is left for the next start.
func RemoveOldExecutables(exe string) {
	dir, name := filepath.Split(exe)
	old := regexp.MustCompile(`^` + regexp.QuoteMeta(name) + `\.old\.\d+$`)
	entries, err := os.ReadDir(filepath.Clean(dir))
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() && old.MatchString(e.Name()) {
			_ = fsx.RemoveAll(filepath.Join(dir, e.Name()))
		}
	}
}
