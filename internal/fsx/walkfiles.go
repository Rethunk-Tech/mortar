package fsx

import (
	"io/fs"
	"path/filepath"
)

// WalkFiles calls fn with the slash path below root, the full path and the size of every regular file under root, in
// walk order.
func WalkFiles(root string, fn func(rel, path string, size int64) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return fn(filepath.ToSlash(rel), path, info.Size())
	})
}
