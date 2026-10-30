//go:build windows

package smapi

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/Rethunk-Tech/go-gameproc"
)

// installerTimeout bounds a run: after an error the installer waits for a key press on its hidden console, which nobody
// can give, so a failed install would otherwise hang.
var installerTimeout = 2 * time.Minute

// start hands the installer's own arguments to PowerShell through the environment, so no quoting rules of ours meet
// PowerShell's. An argument with a space is quoted, its trailing backslashes doubled so the closing quote stays a
// quote. Start-Process gives the child a hidden console of its own; Go would give it NUL handles instead. Reading
// $p.Handle at once keeps the exit code, which is lost when the child exits before anything holds its handle.
const start = `$a = @(); $i = 0; while (Test-Path "env:MORTAR_ARG$i") { $v = (Get-Item "env:MORTAR_ARG$i").Value; if ($v -match '[\s"]') { $v = '"' + ($v -replace '(\\+)$', '$1$1') + '"' }; $a += $v; $i++ }
$o = @{ FilePath = $env:MORTAR_EXE; ArgumentList = $a; WindowStyle = 'Hidden'; PassThru = $true }
if ($env:MORTAR_DIR) { $o.WorkingDirectory = $env:MORTAR_DIR }
$p = Start-Process @o
$null = $p.Handle
if (-not $p.WaitForExit($env:MORTAR_TIMEOUT_MS)) { $p.Kill(); exit 1 }
exit $p.ExitCode`

// runInstaller runs the installer on a hidden console of its own. It clears the screen as it starts, which fails with
// "the handle is invalid" when its output is a pipe, so its output cannot be captured here. PowerShell runs in a job
// object that holds the installer too, so a cancel ends both rather than leaving the installer hidden and waiting.
func runInstaller(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	env := append(os.Environ(), "MORTAR_EXE="+cmd.Path, "MORTAR_DIR="+cmd.Dir,
		"MORTAR_TIMEOUT_MS="+strconv.Itoa(int(installerTimeout.Milliseconds())))
	for i, arg := range cmd.Args[1:] {
		env = append(env, "MORTAR_ARG"+strconv.Itoa(i)+"="+arg)
	}
	ps := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", start)
	ps.Env = env
	gameproc.PrepareTree(ps, true)
	ps.Cancel = func() error {
		gameproc.KillTree(ps.Process.Pid)
		return nil
	}
	if err := ps.Start(); err != nil {
		return nil, err
	}
	pid := ps.Process.Pid
	defer gameproc.UntrackTree(pid)
	// PowerShell starts suspended and runs once the job holds it, so the installer it starts is always born into the job.
	if err := gameproc.TrackTree(pid); err != nil {
		gameproc.KillTree(pid)
		return nil, errors.Join(err, ps.Wait())
	}
	return nil, ps.Wait()
}
