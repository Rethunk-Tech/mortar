//go:build !windows

package fsx

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Trash moves path into the user's freedesktop trash (XDG_DATA_HOME/Trash), where the file manager can restore it.
// A path on another drive than that folder cannot move there and is refused, left where it is.
func Trash(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		data = filepath.Join(home, ".local", "share")
	}
	files, info := filepath.Join(data, "Trash", "files"), filepath.Join(data, "Trash", "info")
	for _, dir := range []string{files, info} {
		if err := os.MkdirAll(dir, 0o700); err != nil { // #nosec G703 -- the user's own trash, from XDG_DATA_HOME or home
			return err
		}
	}
	body := "[Trash Info]\nPath=" + (&url.URL{Path: abs}).EscapedPath() + "\nDeletionDate=" + time.Now().Format("2006-01-02T15:04:05") + "\n"
	base := filepath.Base(abs)
	for i := 1; ; i++ {
		name := base
		if i > 1 {
			name = base + "." + strconv.Itoa(i)
		}
		record := filepath.Join(info, name+".trashinfo")
		f, err := os.OpenFile(record, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 G703 -- a new record in the user's own trash
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return err
		}
		_, werr := f.WriteString(body)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr == nil {
			werr = os.Rename(abs, filepath.Join(files, name)) // #nosec G703 -- a folder the caller chose, into the user's own trash
		}
		if werr != nil {
			_ = os.Remove(record) // #nosec G703 -- the record just written in the user's own trash
			return fmt.Errorf("moving %s to the trash: %w", abs, werr)
		}
		return nil
	}
}
