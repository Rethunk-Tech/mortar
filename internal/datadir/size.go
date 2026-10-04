package datadir

import (
	"io/fs"
	"path/filepath"
)

// Size returns the total size in bytes of the regular files under dir, counting hard-linked files once. Unreadable entries below dir are
// skipped, so the size is a best-effort total; only a failure to read dir itself is an error.
func Size(dir string) (int64, error) {
	var total int64
	seen := map[fileKey]bool{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return rootError(path, dir, err)
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		if info, infoErr := d.Info(); infoErr == nil {
			if key, linked := linkedKey(info); linked {
				if seen[key] {
					return nil
				}
				seen[key] = true
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// fileKey identifies one inode on one device.
type fileKey [2]uint64

func rootError(path, root string, err error) error {
	if path == root {
		return err
	}
	return nil
}
