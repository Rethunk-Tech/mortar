//go:build windows

package datadir

import (
	"os/exec"
	"syscall"
)

// setArgs hands explorer the path as one quoted argument: Go's own quoting leaves a path with a comma to be read as
// explorer's switch list.
func setArgs(cmd *exec.Cmd, dir string) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer "` + dir + `"`}
}
