// Package sandbox runs host commands from inside a Flatpak. The sandbox cannot see the host's Steam, its processes
// or its autostart folder, so Mortar asks the host through flatpak-spawn (the org.freedesktop.Flatpak bus name).
package sandbox

import (
	"os"
	"os/exec"
	"strings"
)

// AppID is Mortar's Flatpak application id.
const AppID = "tech.rethunk.Mortar"

// Getenv and Output (flatpak-spawn's arguments) are replaced in tests.
var (
	Getenv = os.Getenv
	Output = func(args ...string) ([]byte, error) { return exec.Command("flatpak-spawn", args...).Output() }
)

// InFlatpak reports whether Mortar runs inside a Flatpak sandbox.
func InFlatpak() bool { return Getenv("FLATPAK_ID") != "" }

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

// HostOutput runs name on the host and returns its stdout.
func HostOutput(name string, args ...string) ([]byte, error) {
	_, a := HostArgv("", nil, name, args...)
	return Output(a...)
}

// HostLookPath finds name on the host's PATH.
func HostLookPath(name string) (string, error) {
	out, err := HostOutput("which", name)
	if err != nil {
		return "", exec.ErrNotFound
	}
	return strings.TrimSpace(string(out)), nil
}
