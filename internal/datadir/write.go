package datadir

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// ReadJSON reads path into v. Missing files yield found=false and leave v unchanged.
func ReadJSON(path string, v any) (found bool, err error) {
	b, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return true, err
	}
	return true, nil
}

// WriteJSON marshals v and replaces path with it through a temp file and rename.
func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return WriteFile(path, b, 0o600)
}

// WriteFile replaces path with data through a temp file and rename, so a crash never leaves it truncated.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return WriteStream(path, perm, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

// WriteStream replaces path with write's output through a temp file, fsync, and rename, so a crash never leaves it
// truncated. write seeing a failure, or returning one, removes the temp file and leaves path unchanged.
func WriteStream(path string, perm os.FileMode, write func(io.Writer) error) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.Remove(f.Name()))
		}
	}()
	if err = write(f); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Chmod(perm); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Sync(); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
