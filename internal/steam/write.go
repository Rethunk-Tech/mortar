package steam

import (
	"errors"
	"os"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func backupBeforeEdit(path string, _ []byte, perm os.FileMode) error {
	backup := path + ".mortar.bak"
	if _, err := os.Stat(backup); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	body, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		// An account that never had the file (no non-Steam game yet) has nothing to restore.
		return nil
	}
	if err != nil {
		return err
	}
	return fsx.WriteFile(backup, body, perm)
}
