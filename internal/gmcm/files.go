package gmcm

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

const (
	captureDir = "gmcm"
	pendingDir = "gmcm-pending"
	resultExt  = ".result.json"
)

var ErrNoCapture = errors.New("gmcm: no capture")

func CapturePath(profileDir, uniqueID string) string {
	return filepath.Join(profileDir, captureDir, uniqueID+".json")
}

func PendingPath(profileDir, uniqueID string) string {
	return filepath.Join(profileDir, pendingDir, uniqueID+".json")
}

func ResultPath(profileDir, uniqueID string) string {
	return filepath.Join(profileDir, pendingDir, uniqueID+resultExt)
}

func ReadCapture(profileDir, uniqueID string) (Capture, error) {
	var out Capture
	err := readJSON(CapturePath(profileDir, uniqueID), &out)
	if errors.Is(err, fs.ErrNotExist) {
		return out, fmt.Errorf("%w: %s", ErrNoCapture, uniqueID)
	}
	if err != nil {
		return out, err
	}
	return out, checkSchema(out.Schema)
}

func ReadPending(profileDir, uniqueID string) (Pending, error) {
	var out Pending
	err := readJSON(PendingPath(profileDir, uniqueID), &out)
	if errors.Is(err, fs.ErrNotExist) {
		return Pending{Schema: Schema}, nil
	}
	if err != nil {
		return out, err
	}
	return out, checkSchema(out.Schema)
}

func WritePending(profileDir, uniqueID string, edits []Edit) error {
	if err := os.MkdirAll(filepath.Join(profileDir, pendingDir), 0o750); err != nil {
		return err
	}
	if len(edits) == 0 {
		if err := os.Remove(PendingPath(profileDir, uniqueID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	return datadir.WriteJSON(PendingPath(profileDir, uniqueID), Pending{Schema: Schema, Edits: edits})
}

func ReadResult(profileDir, uniqueID string) (Result, error) {
	var out Result
	err := readJSON(ResultPath(profileDir, uniqueID), &out)
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
