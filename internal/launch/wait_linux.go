//go:build linux

package launch

import (
	"os"
	"syscall"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// WaitPID waits until pid exits and reports how. The exit status is only readable for Mortar's own child
// processes; for any other process it waits for the exit and returns the wait error with an empty Exit.
func WaitPID(pid int) (Exit, error) {
	if sandbox.InFlatpak() {
		// Host PIDs mean nothing in the sandbox's PID namespace.
		for Alive(pid) {
			time.Sleep(time.Second)
		}
		return Exit{}, nil
	}
	proc, err := os.FindProcess(pid)
	if err == nil {
		state, werr := proc.Wait()
		if werr == nil {
			if ws, ok := state.Sys().(syscall.WaitStatus); ok {
				return waitStatusExit(ws), nil
			}
			return Exit{Code: state.ExitCode()}, nil
		}
		err = werr
	}
	for Alive(pid) {
		time.Sleep(100 * time.Millisecond)
	}
	return Exit{}, err
}
