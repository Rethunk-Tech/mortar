//go:build windows

package launch

import (
	"time"

	"golang.org/x/sys/windows"
)

// Terminate ends the process at once; Windows has no polite signal for a GUI game, so grace is unused.
func Terminate(pid int, _ time.Duration) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	return windows.TerminateProcess(h, 1)
}
