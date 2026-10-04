//go:build windows

package store

import (
	"errors"
	"os"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// syncPath flushes a file to disk. FlushFileBuffers needs a handle with write access, so a read-only
// open fails with "Access is denied"; directories cannot be flushed this way and NTFS journals their
// metadata, so they are skipped.
func syncPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	f, err := fsx.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}
