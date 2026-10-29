package steam

import (
	"errors"
	"os"

	"github.com/Rethunk-Tech/go-safefs"
)

func backupBeforeEdit(path string, _ []byte, perm os.FileMode) error {
	backup := path + ".mortar.bak"
	if _, err := os.Stat(backup); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	body, err := safefs.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		// An account that never had the file (no non-Steam game yet) has nothing to restore.
		return nil
	}
	if err != nil {
		return err
	}
	return safefs.WriteFile(backup, body, perm)
}
