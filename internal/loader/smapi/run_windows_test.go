//go:build windows

package smapi

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
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

func TestCancelEndsTheInstallerToo(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	t.Setenv("MORTAR_TEST_OUT", pidFile)
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", "$PID | Out-File -Encoding ascii $env:MORTAR_TEST_OUT; Start-Sleep 60")
	if _, err := runInstaller(ctx, cmd); err == nil {
		t.Fatal("a cancelled installer reported success")
	}
	b, err := os.ReadFile(filepath.Clean(pidFile))
	if err != nil {
		t.Fatalf("the installer never started: %v", err)
	}
	pid32, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid32))
	if err != nil {
		return
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if ev, _ := windows.WaitForSingleObject(h, 5000); ev != windows.WAIT_OBJECT_0 {
		t.Fatalf("installer %d still runs after the cancel", pid32)
	}
}

func TestQuotedArgumentKeepsItsTrailingBackslash(t *testing.T) {
	out := filepath.Join(t.TempDir(), "arg")
	dir := t.TempDir() + `\with space\`
	t.Setenv("MORTAR_TEST_OUT", out)
	// Built as a struct, not exec.Command: runInstaller reads only Path, Dir and Args.
	cmd := &exec.Cmd{Path: "powershell.exe", Args: []string{"powershell.exe", "-NoProfile", "-File", writeEcho(t), dir}}
	if _, err := runInstaller(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Clean(out))
	if err != nil || strings.TrimSpace(string(b)) != dir {
		t.Fatalf("the installer got %q, want %q (%v)", b, dir, err)
	}
}

// writeEcho is a script that writes its first argument to $env:MORTAR_TEST_OUT.
func writeEcho(t *testing.T) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "echo.ps1")
	if err := os.WriteFile(script, []byte("$args[0] | Out-File -Encoding ascii $env:MORTAR_TEST_OUT"), 0o600); err != nil {
		t.Fatal(err)
	}
	return script
}
