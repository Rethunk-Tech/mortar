//go:build windows

package launch

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// hideWindow starts a console program without a console. SW_HIDE alone is ignored when Windows 11 hands the new
// console to Windows Terminal, the default terminal, so the window would still show; CREATE_NO_WINDOW never makes one.
// SMAPI's output still reaches Mortar through the captured pipes and its own log file.
func hideWindow(cmd *exec.Cmd, hide bool) {
	if !hide {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}
