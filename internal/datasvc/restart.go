package datasvc

import (
	"context"
	"os"
	"os/exec"
)

// RestartSelf starts this executable again, without the old arguments (a fresh start is what a data move needs, and
// they may name a link already handled), and exits.
func RestartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(context.Background(), exe)
	cmd.Env = os.Environ()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
