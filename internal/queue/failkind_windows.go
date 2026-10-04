package queue

import (
	"errors"

	"golang.org/x/sys/windows"
)

// An antivirus scanner shows up as a virus error or as access denied on a file Mortar just wrote.
func platformBlocked(err error) bool {
	return errors.Is(err, windows.ERROR_VIRUS_INFECTED) ||
		errors.Is(err, windows.ERROR_VIRUS_DELETED) ||
		errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
