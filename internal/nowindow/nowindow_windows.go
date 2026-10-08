// Package nowindow starts console programs from Mortar's GUI process without a console window.
package nowindow

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// Set makes cmd start without a console window. Mortar has no console of its own, so Windows otherwise opens one
// for every console program it starts (powershell, netsh, gh). CREATE_NO_WINDOW, unlike SW_HIDE, is also honoured
// when Windows Terminal is the default console.
func Set(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NO_WINDOW
}
