//go:build windows

package launch

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

// stillActive is GetExitCodeProcess's code for a process that has not exited.
const stillActive = 259

// Terminate ends the process at once, since Windows has no polite signal for a GUI game, and waits up to grace
// for it to be gone: TerminateProcess returns before the game's files are released.
func Terminate(pid int, grace time.Duration) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if err := windows.TerminateProcess(h, 1); err != nil {
		return err
	}
	ev, err := windows.WaitForSingleObject(h, uint32(grace.Milliseconds()))
	if err != nil {
		return err
	}
	if ev != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("process %d is still exiting after %s", pid, grace)
	}
	return nil
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
