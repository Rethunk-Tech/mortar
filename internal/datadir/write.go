package datadir

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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

// WriteJSONIn is WriteJSON to name inside dir, making dir first.
func WriteJSONIn(dir, name string, v any) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return WriteJSON(filepath.Join(dir, name), v)
}

// WriteOrRemove writes data to path, making its folder, or removes path when data is empty.
func WriteOrRemove(path string, data []byte) error {
	if len(data) == 0 {
		if err := fsx.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return WriteFile(path, data, 0o600)
}

// ReadJournal reads the JSON journal at path into v unless the file is missing or busy reports that it must be left
// alone for now; found says whether v was read.
func ReadJournal(path string, busy func() bool, v any) (found bool, err error) {
	b, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if busy != nil && busy() {
		return false, nil
	}
	return true, json.Unmarshal(b, v)
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
func WriteStream(path string, perm os.FileMode, write func(io.Writer) error) error {
	return writeStream(path, perm, true, write)
}

// WriteCacheStream is WriteStream without the fsyncs, for a file that only memoises a computation: a crash may
// leave it torn, so the reader must treat an unreadable file as absent. Thousands of small fsyncs cost seconds.
func WriteCacheStream(path string, perm os.FileMode, write func(io.Writer) error) error {
	return writeStream(path, perm, false, write)
}

// syncWrites is off under go test: a test cannot observe an fsync, and the thousands its fixtures write cost the
// suite seconds on a real disk.
var syncWrites = !testing.Testing()

func writeStream(path string, perm os.FileMode, durable bool, write func(io.Writer) error) (err error) {
	durable = durable && syncWrites
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, fsx.Remove(f.Name()))
		}
	}()
	if err = write(f); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Chmod(perm); err != nil {
		return errors.Join(err, f.Close())
	}
	if durable {
		if err = f.Sync(); err != nil {
			return errors.Join(err, f.Close())
		}
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = fsx.Rename(f.Name(), path); err != nil {
		return err
	}
	if durable {
		_ = fsx.SyncDir(filepath.Dir(path)) // best effort: the rename already landed
	}
	return nil
}
