//go:build windows

package datadir

import "golang.org/x/sys/windows"

// The errors a clone, hard link and symlink refuse with when the target cannot do them.
var errNoClone, errCrossDevice, errBadLink error = windows.ERROR_NOT_SUPPORTED, windows.ERROR_NOT_SAME_DEVICE, windows.ERROR_INVALID_PARAMETER
