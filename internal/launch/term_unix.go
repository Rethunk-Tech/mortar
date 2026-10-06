//go:build !windows

package launch

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// Stop ends a game process. A Wine game is first asked to close its windows, as quitting it would: one that registered
// with a running Steam signs off only on a clean exit, and killed instead it stays listed under its Wine pid, which
// Steam then waits on for good. Whatever is still running after grace is terminated.
func Stop(ctx context.Context, p Process, grace time.Duration) error {
	if closeWine(ctx, p, grace) {
		deadline := time.Now().Add(grace)
		for time.Now().Before(deadline) {
			if !Alive(p.PID) {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return Terminate(p.PID, grace)
}

// closeWine runs taskkill without /f in the process's own Wine environment, which posts WM_CLOSE to the windows of
// every process with that image name in its prefix. It reports whether taskkill ran.
func closeWine(ctx context.Context, p Process, timeout time.Duration) bool {
	wine := wineLoader(p.Exe)
	if wine == "" || len(p.Args) == 0 || sandbox.InFlatpak() {
		return false
	}
	image := filepath.Base(strings.ReplaceAll(p.Args[0], `\`, "/"))
	if !strings.HasSuffix(strings.ToLower(image), ".exe") {
		return false
	}
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(p.PID), "environ"))
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, wine, "taskkill", "/im", image)
	// WINESERVERSOCKET names a descriptor the game inherited, which taskkill does not have.
	for kv := range strings.SplitSeq(strings.TrimRight(string(raw), "\x00"), "\x00") {
		if !strings.HasPrefix(kv, "WINESERVERSOCKET=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	return cmd.Run() == nil
}

// wineLoader is the wine launcher beside exe when exe is a Wine loader (Proton keeps both in one folder), or "".
func wineLoader(exe string) string {
	exe = strings.TrimSuffix(exe, " (deleted)")
	switch filepath.Base(exe) {
	case "wine", "wine64", "wine-preloader", "wine64-preloader":
	default:
		return ""
	}
	for _, name := range []string{"wine", "wine64"} {
		path := filepath.Join(filepath.Dir(exe), name)
		if st, err := os.Stat(path); err == nil && st.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

// Terminate asks the process to exit and kills it if it is still there after grace.
func Terminate(pid int, grace time.Duration) error {
	if sandbox.InFlatpak() {
		return hostTerminate(pid, grace)
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// Alive reports whether a process with this pid exists.
func Alive(pid int) bool {
	if sandbox.InFlatpak() {
		return hostAlive(pid)
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// Host PIDs mean nothing in the sandbox's PID namespace, so a Flatpak signals through the host's kill.
func hostKill(sig string, pid int) error {
	_, err := sandbox.HostOutput("kill", "-"+sig, strconv.Itoa(pid))
	return err
}

func hostAlive(pid int) bool { return hostKill("0", pid) == nil }

func hostTerminate(pid int, grace time.Duration) error {
	if !hostAlive(pid) {
		return nil
	}
	if hostKill("TERM", pid) != nil && !hostAlive(pid) {
		return nil
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !hostAlive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = hostKill("KILL", pid)
	return nil
}
