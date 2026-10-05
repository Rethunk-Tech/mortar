//go:build windows

package smapi

import (
	"context"
	"os/exec"
	"testing"
)

func TestRunInstallerGivesTheChildAConsoleItCanClear(t *testing.T) {
	args := []string{"-NoProfile", "-Command", "[Console]::Clear(); exit 0"}
	if _, err := runInstaller(context.Background(), exec.CommandContext(context.Background(), "powershell.exe", args...)); err != nil {
		t.Fatalf("clearing the console failed: %v", err)
	}
}

func TestRunInstallerReportsTheChildsFailure(t *testing.T) {
	args := []string{"-NoProfile", "-Command", "exit 3"}
	if _, err := runInstaller(context.Background(), exec.CommandContext(context.Background(), "powershell.exe", args...)); err == nil {
		t.Fatal("a failing installer reported success")
	}
}
