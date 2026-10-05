//go:build !windows

package smapi

import (
	"context"
	"os/exec"
)

// runInstaller runs the installer and returns what it printed, which carries its error.
func runInstaller(ctx context.Context, exe, dir string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
