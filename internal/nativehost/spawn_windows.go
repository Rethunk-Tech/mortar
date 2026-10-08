//go:build windows

package nativehost

import (
	"context"
	"errors"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// start detaches Mortar from the browser's job: Firefox runs a native host in a kill-on-close job, which would end
// Mortar with the host. A job that forbids breakaway refuses the flag, so the spawn is retried without it.
func start(exe, link string) error {
	err := spawn(exe, link, windows.CREATE_BREAKAWAY_FROM_JOB)
	if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
		err = spawn(exe, link, 0)
	}
	return err
}

func spawn(exe, link string, extra uint32) error {
	cmd := exec.CommandContext(context.Background(), exe, link)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS | extra}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
