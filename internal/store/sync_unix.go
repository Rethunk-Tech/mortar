//go:build !windows

package store

import (
	"errors"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// syncPath flushes a file or directory to disk; a read-only handle is enough here.
func syncPath(path string) error {
	f, err := fsx.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}
