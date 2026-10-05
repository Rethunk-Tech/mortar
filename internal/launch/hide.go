//go:build !windows

package launch

import (
	"os/exec"
	"syscall"
)

// hideWindow puts the process in its own group, so a stalled launch can be ended with everything it started.
func hideWindow(cmd *exec.Cmd, _ bool) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
