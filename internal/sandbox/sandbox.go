// Package sandbox runs host commands from inside a Flatpak. The sandbox cannot see the host's Steam, its processes
// or its autostart folder, so Mortar asks the host through flatpak-spawn (the org.freedesktop.Flatpak bus name).
package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Getenv and Output (flatpak-spawn's arguments) are replaced in tests.
var (
	Getenv = os.Getenv
	Output = func(args ...string) ([]byte, error) {
		return exec.Command("flatpak-spawn", args...).Output() //nolint:gosec // G204: binary is flatpak-spawn
	}
)

// InFlatpak reports whether Mortar runs inside a Flatpak sandbox.
func InFlatpak() bool { return Getenv("FLATPAK_ID") != "" }

// HostConfigHome is the host user's config home: XDG_CONFIG_HOME when set and absolute, else ~/.config. Inside a
// Flatpak, XDG_CONFIG_HOME is the app's own folder and Flatpak passes the host's custom one as HOST_XDG_CONFIG_HOME,
// where it mounts the filesystem grants.
func HostConfigHome(home string) string {
	return hostXDG("XDG_CONFIG_HOME", filepath.Join(homeOr(home), ".config"))
}

// HostDataHome is HostConfigHome for the data home (XDG_DATA_HOME).
func HostDataHome(home string) string {
	return hostXDG("XDG_DATA_HOME", filepath.Join(homeOr(home), ".local", "share"))
}

// homeOr is home, or the user's own home when none was given: an empty home would make the fallback a path relative
// to the working directory.
func homeOr(home string) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return home
}

func hostXDG(name, fallback string) string {
	if InFlatpak() {
		name = "HOST_" + name
	}
	if v := Getenv(name); filepath.IsAbs(v) {
		return v
	}
	return fallback
}

// HostArgv returns the command line that runs name on the host with dir as its working directory and env added.
func HostArgv(dir string, env []string, name string, args ...string) (string, []string) {
	out := []string{"--host"}
	if dir != "" {
		out = append(out, "--directory="+dir)
	}
	for _, e := range env {
		out = append(out, "--env="+e)
	}
	out = append(out, name)
	return "flatpak-spawn", append(out, args...)
}

// cancelGrace is how long a cancelled flatpak-spawn gets to forward SIGTERM and exit before Go gives up on it.
const cancelGrace = 5 * time.Second

// CommandContext is exec.CommandContext that ends a flatpak-spawn host command on cancel. flatpak-spawn forwards
// SIGTERM to the host child but cannot forward the SIGKILL Go sends by default, which would leave the child running.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // G204: callers pass a resolved binary or flatpak-spawn
	if name == "flatpak-spawn" {
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
		cmd.WaitDelay = cancelGrace
	}
	return cmd
}

// HostOutput runs name on the host and returns its stdout.
func HostOutput(name string, args ...string) ([]byte, error) {
	_, a := HostArgv("", nil, name, args...)
	return Output(a...)
}

// HostLookPath finds name on the host's PATH. It asks the host's sh rather than which, which minimal hosts (Arch's
// base) do not install.
func HostLookPath(name string) (string, error) {
	out, err := HostOutput("sh", "-c", `command -v -- "$1"`, "sh", name)
	if err != nil {
		return "", exec.ErrNotFound
	}
	// command -v prints a builtin or alias by its bare name, which is not a path.
	p := strings.TrimSpace(string(out))
	if !filepath.IsAbs(p) {
		return "", exec.ErrNotFound
	}
	return p, nil
}
