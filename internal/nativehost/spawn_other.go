//go:build !windows

package nativehost

import (
	"context"
	"os/exec"
)

func start(exe, link string) error {
	cmd := exec.CommandContext(context.Background(), exe, link)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
