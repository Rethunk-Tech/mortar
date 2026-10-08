//go:build windows

package launch

import (
	"context"
	"fmt"
	"math"
	"time"

	"golang.org/x/sys/windows"
)

// stillActive is GetExitCodeProcess's code for a process that has not exited.
const stillActive = 259

func pidUint32(pid int) (uint32, error) {
	if pid < 0 || uint64(pid) > math.MaxUint32 {
		return 0, fmt.Errorf("pid %d out of range", pid)
	}
	return uint32(pid), nil
}

func waitMillis(d time.Duration) uint32 {
	ms := d.Milliseconds()
	if ms < 0 {
		return 0
	}
	if ms > int64(math.MaxUint32) {
		return math.MaxUint32
	}
	return uint32(ms)
}

// Stop ends a game process; see Terminate.
func Stop(_ context.Context, p Process, grace time.Duration) error { return Terminate(p.PID, grace) }

// Terminate ends the process at once, since Windows has no polite signal for a GUI game, and waits up to grace
// for it to be gone: TerminateProcess returns before the game's files are released.
func Terminate(pid int, grace time.Duration) error {
	id, err := pidUint32(pid)
	if err != nil {
		return err
	}
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, id)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if !terminateTracked(pid) {
		if err := windows.TerminateProcess(h, 1); err != nil {
			return err
		}
	}
	ev, err := windows.WaitForSingleObject(h, waitMillis(grace))
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
	id, err := pidUint32(pid)
	if err != nil {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, id)
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var code uint32
	return windows.GetExitCodeProcess(h, &code) == nil && code == stillActive
}
