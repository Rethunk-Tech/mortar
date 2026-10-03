package datadir

import (
	"io/fs"
	"path/filepath"
)

// Size returns the total size in bytes of the regular files under dir. Unreadable entries below dir are
// skipped, so the size is a best-effort total; only a failure to read dir itself is an error.
func Size(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return rootError(path, dir, err)
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		if info, infoErr := d.Info(); infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func rootError(path, root string, err error) error {
	if path == root {
		return err
	}
	return nil
}
