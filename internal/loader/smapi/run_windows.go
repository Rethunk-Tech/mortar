//go:build windows

package smapi

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// start hands the installer's own arguments to PowerShell through the environment, so no quoting rules of ours meet
// PowerShell's. Start-Process gives the child a hidden console of its own; Go would give it NUL handles instead.
const start = `$a = @(); $i = 0; while (Test-Path "env:MORTAR_ARG$i") { $v = (Get-Item "env:MORTAR_ARG$i").Value; if ($v -match '[\s"]') { $v = '"' + $v + '"' }; $a += $v; $i++ }
$p = Start-Process -FilePath $env:MORTAR_EXE -ArgumentList $a -WorkingDirectory $env:MORTAR_DIR -WindowStyle Hidden -Wait -PassThru
exit $p.ExitCode`

// runInstaller runs the installer on a hidden console of its own. It clears the screen as it starts, which fails with
// "the handle is invalid" when its output is a pipe, so its output cannot be captured here.
func runInstaller(ctx context.Context, exe, dir string, args []string) ([]byte, error) {
	env := append(os.Environ(), "MORTAR_EXE="+exe, "MORTAR_DIR="+dir)
	for i, arg := range args {
		env = append(env, "MORTAR_ARG"+strconv.Itoa(i)+"="+arg)
	}
	ps := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", start)
	ps.Env = env
	ps.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return nil, ps.Run()
}
