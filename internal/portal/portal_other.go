//go:build !linux

package portal

import "errors"

var errUnsupported = errors.New("desktop portals exist only on Linux")

// ErrCancelled is returned when the user dismisses the portal's dialog.
var ErrCancelled = errors.New("cancelled in the system dialog")

func SetAutostart(bool, []string) error                    { return errUnsupported }
func InstallLauncher(string, string, string, []byte) error { return errUnsupported }
func UninstallLauncher(string) error                       { return errUnsupported }
func LauncherExists(string) (bool, error)                  { return false, errUnsupported }
