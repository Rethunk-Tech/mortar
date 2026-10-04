//go:build !windows

package launch

import (
	"errors"
	"strconv"
	"syscall"
	"time"

	"github.com/Rethunk-AI/mortar/internal/sandbox"
)

// Terminate asks the process to exit and kills it if it is still there after grace.
func Terminate(pid int, grace time.Duration) error {
	if sandbox.InFlatpak() {
		return hostTerminate(pid, grace)
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// Alive reports whether a process with this pid exists.
func Alive(pid int) bool {
	if sandbox.InFlatpak() {
		return hostAlive(pid)
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// Host PIDs mean nothing in the sandbox's PID namespace, so a Flatpak signals through the host's kill.
func hostKill(sig string, pid int) error {
	_, err := sandbox.HostOutput("kill", "-"+sig, strconv.Itoa(pid))
	return err
}

func hostAlive(pid int) bool { return hostKill("0", pid) == nil }

func hostTerminate(pid int, grace time.Duration) error {
	if !hostAlive(pid) {
		return nil
	}
	if hostKill("TERM", pid) != nil && !hostAlive(pid) {
		return nil
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !hostAlive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = hostKill("KILL", pid)
	return nil
}
