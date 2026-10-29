package folderwatch

import (
	"errors"

	"golang.org/x/sys/windows"
)

// held reports whether another process has name open for writing: an exclusive open then fails with a sharing
// violation. Windows delays the write notifications of a file that stays open, so a download written in place
// shows only its create event until it closes.
func held(name string) bool {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return false
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return errors.Is(err, windows.ERROR_SHARING_VIOLATION)
	}
	_ = windows.CloseHandle(h)
	return false
}
