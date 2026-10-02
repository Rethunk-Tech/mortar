package nativehost

import (
	"context"
	"os/exec"
)

// Start runs Mortar at exe on link without waiting: the new process forwards the link to the running Mortar, or
// becomes it.
func Start(exe, link string) error {
	cmd := exec.CommandContext(context.Background(), exe, link)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
