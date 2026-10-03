package gmcm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const (
	captureDir = "gmcm"
	pendingDir = "gmcm-pending"
	indexName  = "_index.json"
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

func ReadIndex(profileDir string) (Index, error) {
	var out Index
	err := readJSON(filepath.Join(profileDir, captureDir, indexName), &out)
	if err != nil {
		return out, err
	}
	return out, checkSchema(out.Schema)
}

func CapturedIDs(profileDir string) ([]string, error) {
	idx, err := ReadIndex(profileDir)
	if err == nil && len(idx.Mods) > 0 {
		ids := make([]string, 0, len(idx.Mods))
		for _, m := range idx.Mods {
			if m.ID != "" {
				ids = append(ids, m.ID)
			}
		}
		return ids, nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(profileDir, captureDir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == indexName || !strings.HasSuffix(name, ".json") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".json"))
	}
	return ids, nil
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
	return writeJSONAtomic(PendingPath(profileDir, uniqueID), Pending{Schema: Schema, Edits: edits})
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
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func writeJSONAtomic(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gmcm-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
