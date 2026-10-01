package datasvc

import (
	"context"
	"os"
	"os/exec"
)

// RestartSelf starts this executable again and exits.
func RestartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(context.Background(), exe, os.Args[1:]...) //nolint:gosec // Mortar restarts itself after moving the data folder
	cmd.Env = os.Environ()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
