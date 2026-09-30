//go:build windows

package launch

import (
	"time"

	"golang.org/x/sys/windows"
)

// stillActive is GetExitCodeProcess's code for a process that has not exited.
const stillActive = 259

// Terminate ends the process at once; Windows has no polite signal for a GUI game, so grace is unused.
func Terminate(pid int, _ time.Duration) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	return windows.TerminateProcess(h, 1)
}

// Alive reports whether a process with this pid is still running.
func Alive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var code uint32
	return windows.GetExitCodeProcess(h, &code) == nil && code == stillActive
}
