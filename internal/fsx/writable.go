package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// CheckWritable fails fast when dir refuses new files, as a game under Program Files does for a non-elevated
// process, instead of letting the first real write fail halfway through an install.
func CheckWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".mortar-probe-*")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // creating the folder is the caller's job
		}
		if errors.Is(err, fs.ErrPermission) {
			return usererr.Wrap(usererr.Permission, fmt.Errorf(
				"cannot write to %s: move the game out of Program Files, or fix the folder's permissions (the store's repair or verify option can reset them): %w", dir, err))
		}
		return err
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}
