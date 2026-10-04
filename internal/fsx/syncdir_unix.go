//go:build !windows

package fsx

import "errors"

// SyncDir flushes a directory's entries (a rename into it) to disk.
func SyncDir(dir string) error {
	f, err := Open(dir)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}
