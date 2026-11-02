//go:build windows

package devsvc

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// toolCommand runs name directly, or through cmd.exe with its own quoting when name resolves to a command file: dotnet
// tools and VS Code install one, and Go's quoting leaves cmd.exe splitting its path at a space.
func toolCommand(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		path = name
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cmd" && ext != ".bat" {
		return exec.CommandContext(ctx, name, args...), nil // #nosec G204 -- fixed tools; arguments validated by the caller
	}
	system, err := windows.GetSystemDirectory()
	if err != nil {
		return nil, err
	}
	shell := filepath.Join(system, "cmd.exe")
	line, err := batchCommandLine(shell, path, args)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, shell) //nolint:gosec // G204: the system folder's own cmd.exe; arguments are refused or quoted by batchCommandLine
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: line}
	return cmd, nil
}
