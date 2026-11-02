//go:build !windows

package devsvc

import (
	"context"
	"os/exec"
)

func toolCommand(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	return exec.CommandContext(ctx, name, args...), nil // #nosec G204 -- fixed tools; arguments validated by the caller
}
