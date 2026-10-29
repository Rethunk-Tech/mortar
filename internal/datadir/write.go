package datadir

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

// ReadJSON reads path into v. Missing files yield found=false and leave v unchanged.
func ReadJSON(path string, v any) (found bool, err error) {
	b, err := safefs.ReadFile(path)
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

// ReadJSONOrQuarantine is ReadJSON for a file whose loss is cheaper than a Mortar that will not start: an unreadable
// JSON file is moved beside itself as <path>.corrupt-<unixnano> and v is left for the caller to reset.
func ReadJSONOrQuarantine(path string, v any) (found bool, err error) {
	found, err = ReadJSON(path, v)
	var syn *json.SyntaxError
	var typ *json.UnmarshalTypeError
	if !found || err == nil || (!errors.As(err, &syn) && !errors.As(err, &typ) && !errors.Is(err, io.ErrUnexpectedEOF)) {
		return found, err
	}
	aside := fmt.Sprintf("%s.corrupt-%d", path, time.Now().UnixNano())
	if rerr := safefs.Rename(path, aside); rerr != nil {
		return true, err
	}
	slog.Warn("moved an unreadable file aside", "file", path, "to", aside, "error", err)
	return false, nil
}

// WriteJSON marshals v and replaces path with it through a temp file and rename.
func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return safefs.WriteFileAtomic(path, b, 0o600)
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
		if err := safefs.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return safefs.WriteFileAtomic(path, data, 0o600)
}

// ReadJournal reads the JSON journal at path into v unless the file is missing or busy reports that it must be left
// alone for now; found says whether v was read.
func ReadJournal(path string, busy func() bool, v any) (found bool, err error) {
	b, err := safefs.ReadFile(path)
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
