//go:build windows

package launch

import (
	"os/exec"
	"syscall"
)

func hideWindow(cmd *exec.Cmd, hide bool) {
	if !hide {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
