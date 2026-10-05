package gmcm

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

const (
	captureDir = "gmcm"
	pendingDir = "gmcm-pending"
	resultExt  = ".result.json"
)

var ErrNoCapture = errors.New("gmcm: no capture")

// fileName is the id's local part: the in-game bridge writes these files and names them by SMAPI's unique id.
func fileName(id mod.ID) string { return id.Local() }

func CapturePath(profileDir string, id mod.ID) string {
	return filepath.Join(profileDir, captureDir, fileName(id)+".json")
}

func PendingPath(profileDir string, id mod.ID) string {
	return filepath.Join(profileDir, pendingDir, fileName(id)+".json")
}

func ResultPath(profileDir string, id mod.ID) string {
	return filepath.Join(profileDir, pendingDir, fileName(id)+resultExt)
}

func ReadCapture(profileDir string, id mod.ID) (Capture, error) {
	var out Capture
	err := readJSON(CapturePath(profileDir, id), &out)
	if errors.Is(err, fs.ErrNotExist) {
		return out, fmt.Errorf("%w: %s", ErrNoCapture, id)
	}
	if err != nil {
		return out, err
	}
	return out, checkSchema(out.Schema)
}

func ReadPending(profileDir string, id mod.ID) (Pending, error) {
	var out Pending
	err := readJSON(PendingPath(profileDir, id), &out)
	if errors.Is(err, fs.ErrNotExist) {
		return Pending{Schema: Schema}, nil
	}
	if err != nil {
		return out, err
	}
	return out, checkSchema(out.Schema)
}

func WritePending(profileDir string, id mod.ID, edits []Edit) error {
	if err := os.MkdirAll(filepath.Join(profileDir, pendingDir), 0o750); err != nil {
		return err
	}
	if len(edits) == 0 {
		if err := os.Remove(PendingPath(profileDir, id)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	return datadir.WriteJSON(PendingPath(profileDir, id), Pending{Schema: Schema, Edits: edits})
}

func ReadResult(profileDir string, id mod.ID) (Result, error) {
	var out Result
	err := readJSON(ResultPath(profileDir, id), &out)
	if errors.Is(err, fs.ErrNotExist) {
		return Result{}, nil
	}
	if err != nil {
		return out, err
	}
	return out, checkSchema(out.Schema)
}

func checkSchema(got int) error {
	if got != Schema {
		return fmt.Errorf("gmcm: unsupported schema %d (want %d)", got, Schema)
	}
	return nil
}

func readJSON(path string, dest any) error {
	found, err := datadir.ReadJSON(path, dest)
	if err != nil {
		return err
	}
	if !found {
		return fs.ErrNotExist
	}
	return nil
}
