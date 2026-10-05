//go:build !windows

package smapi

import (
	"context"
	"os/exec"
)

// runInstaller runs the installer and returns what it printed, which carries its error.
func runInstaller(_ context.Context, cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}
