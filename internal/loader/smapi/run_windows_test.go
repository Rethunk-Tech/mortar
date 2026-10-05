//go:build windows

package smapi

import (
	"context"
	"os/exec"
	"testing"
	"time"
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

func TestRunInstallerGivesUpOnAnInstallerWaitingForAKey(t *testing.T) {
	installerTimeout = 3 * time.Second
	t.Cleanup(func() { installerTimeout = 2 * time.Minute })
	args := []string{"-NoProfile", "-Command", "Start-Sleep 60"}
	start := time.Now()
	if _, err := runInstaller(context.Background(), exec.CommandContext(context.Background(), "powershell.exe", args...)); err == nil || time.Since(start) > 30*time.Second {
		t.Fatalf("a stuck installer was not stopped (err %v after %v)", err, time.Since(start))
	}
}
