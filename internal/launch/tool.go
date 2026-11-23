package launch

import (
	"context"
	"errors"
	"os/exec"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-gameproc"
)

// RunTool runs a helper program of the game to its end with no window and returns its exit code and combined output.
// A program that ends with a non-zero code is a result; only one that could not run at all is an error.
func RunTool(ctx context.Context, env []string, dir, name string, args ...string) (int, string, error) {
	cmd := flatpak.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = commandEnv(dir, env)
	gameproc.PrepareTree(cmd, true)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), string(out), nil
	}
	return 0, string(out), err
}
