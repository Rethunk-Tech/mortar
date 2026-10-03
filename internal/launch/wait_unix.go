//go:build unix

package launch

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// WaitError maps cmd.Wait's result to an Exit.
func WaitError(err error) Exit {
	if err == nil {
		return Exit{Code: 0}
	}
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		x := Exit{Code: ee.ExitCode()}
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			x.Signal = unixSignalName(ws.Signal())
			if x.Code < 0 {
				x.Code = 128 + int(ws.Signal())
			}
		}
		return x
	}
	return Exit{Code: 1}
}

func waitStatusExit(ws syscall.WaitStatus) Exit {
	if ws.Signaled() {
		sig := ws.Signal()
		return Exit{Code: 128 + int(sig), Signal: unixSignalName(sig)}
	}
	return Exit{Code: ws.ExitStatus()}
}

func unixSignalName(sig syscall.Signal) string {
	if name := unix.SignalName(sig); name != "" {
		return name
	}
	s := strings.TrimSpace(sig.String())
	if s == "" {
		return ""
	}
	return s
}
