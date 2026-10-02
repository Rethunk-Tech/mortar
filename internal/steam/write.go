package steam

import (
	"errors"
	"os"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func backupBeforeEdit(path string, _ []byte, perm os.FileMode) error {
	backup := path + ".mortar.bak"
	if _, err := os.Stat(backup); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	body, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	return fsx.WriteFile(backup, body, perm)
}
