package datadir

import (
	"context"
	"os/exec"
	"runtime"
)

// Open shows dir in the system file manager without waiting for it.
func Open(dir string) error {
	name := "xdg-open"
	switch runtime.GOOS {
	case "windows":
		name = "explorer"
	case "darwin":
		name = "open"
	}
	cmd := exec.CommandContext(context.Background(), name, dir)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
