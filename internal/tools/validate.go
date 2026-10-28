package tools

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// refuseWindowsProgram is an error for a .exe on Linux, which cannot run it without Wine or Proton.
func refuseWindowsProgram(exe string) error {
	if runtime.GOOS == "linux" && strings.HasSuffix(strings.ToLower(exe), ".exe") {
		return errors.New("a Windows program (.exe) cannot be started directly on Linux: pick a Linux program or a script that starts it through Wine or Proton")
	}
	return nil
}

func validateTool(t Tool, ctx Context) error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(t.Executable) == "" {
		return fmt.Errorf("executable is required")
	}
	exe := expand(t.Executable, ctx)
	if err := refuseWindowsProgram(exe); err != nil {
		return err
	}
	info, err := os.Stat(exe)
	if err != nil {
		return fmt.Errorf("executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("executable must be a regular file")
	}
	return nil
}
