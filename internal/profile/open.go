package profile

import (
	"context"
	"os/exec"
	"runtime"
)

func openFolder(dir string) error {
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
