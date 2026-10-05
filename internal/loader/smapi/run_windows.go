//go:build windows

package smapi

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// start hands the installer's own arguments to PowerShell through the environment, so no quoting rules of ours meet
// PowerShell's. Start-Process gives the child a hidden console of its own; Go would give it NUL handles instead.
// installerTimeout bounds a run: after an error the installer waits for a key press on its hidden console, which nobody
// can give, so a failed install would otherwise hang.
var installerTimeout = 2 * time.Minute

const start = `$a = @(); $i = 0; while (Test-Path "env:MORTAR_ARG$i") { $v = (Get-Item "env:MORTAR_ARG$i").Value; if ($v -match '[\s"]') { $v = '"' + $v + '"' }; $a += $v; $i++ }
$o = @{ FilePath = $env:MORTAR_EXE; ArgumentList = $a; WindowStyle = 'Hidden'; PassThru = $true }
if ($env:MORTAR_DIR) { $o.WorkingDirectory = $env:MORTAR_DIR }
$p = Start-Process @o
if (-not $p.WaitForExit($env:MORTAR_TIMEOUT_MS)) { $p.Kill(); exit 1 }
exit $p.ExitCode`

// runInstaller runs the installer on a hidden console of its own. It clears the screen as it starts, which fails with
// "the handle is invalid" when its output is a pipe, so its output cannot be captured here.
func runInstaller(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	env := append(os.Environ(), "MORTAR_EXE="+cmd.Path, "MORTAR_DIR="+cmd.Dir,
		"MORTAR_TIMEOUT_MS="+strconv.Itoa(int(installerTimeout.Milliseconds())))
	for i, arg := range cmd.Args[1:] {
		env = append(env, "MORTAR_ARG"+strconv.Itoa(i)+"="+arg)
	}
	ps := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", start)
	ps.Env = env
	ps.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return nil, ps.Run()
}
