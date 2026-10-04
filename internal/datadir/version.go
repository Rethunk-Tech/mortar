package datadir

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

// FormatVersion is the on-disk format this build reads and writes for profile.json, settings.json, queue.json and
// history.json. Those structs carry it as "formatVersion"; a file without one is current data.
const FormatVersion = 1

// CheckVersion is the migration point: v is a file's formatVersion, 0 when the file has none.
func CheckVersion(v int) error {
	switch {
	case v <= FormatVersion:
		return nil
	default:
		return usererr.New(usererr.Invalid, "this file was written by a newer version of Mortar, so this version will not change it; update Mortar")
	}
}

// WriteVersioned writes v like WriteJSON, unless the file on disk was written by a newer Mortar. Then it copies
// that file to path+".newer" once and leaves it untouched. Callers set the struct's FormatVersion to FormatVersion.
func WriteVersioned(path string, v any) error {
	b, err := fsx.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var head struct {
		FormatVersion int `json:"formatVersion"`
	}
	if json.Unmarshal(b, &head) == nil {
		if verr := CheckVersion(head.FormatVersion); verr != nil {
			if _, statErr := os.Stat(path + ".newer"); statErr != nil {
				if werr := WriteFile(path+".newer", b, 0o600); werr != nil {
					return errors.Join(verr, werr)
				}
			}
			return verr
		}
	}
	return WriteJSON(path, v)
}
