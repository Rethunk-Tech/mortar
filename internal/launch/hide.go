//go:build !windows

package launch

import (
	"os/exec"
	"syscall"
)

// hideWindow puts the process in its own group, so a stalled launch can be ended with everything it started.
func hideWindow(cmd *exec.Cmd, _ bool) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// ownsConsole is false: on Linux and macOS the process's input and output always go through Mortar.
func ownsConsole(bool) bool { return false }
