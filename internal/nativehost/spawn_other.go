//go:build !windows

package nativehost

import (
	"context"
	"os/exec"

	"github.com/Rethunk-Tech/mortar/internal/hostenv"
)

func start(exe, link string) error {
	cmd := exec.CommandContext(context.Background(), exe, link) //nolint:gosec // G204: exe is Mortar; link is a mortar:// URL
	cmd.Env = hostenv.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
