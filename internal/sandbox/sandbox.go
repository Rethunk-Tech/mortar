// Package sandbox builds the bubblewrap command line that starts a direct game launch with the user's home hidden, so
// code in a mod can reach only the folders the game needs.
package sandbox

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

// ErrNoBwrap means the opt-in sandbox is on but bubblewrap is not installed.
var ErrNoBwrap = errors.New("the game sandbox needs bubblewrap (bwrap), which is not installed")

// Bwrap is the path of the bubblewrap program.
func Bwrap() (string, error) {
	p, err := exec.LookPath("bwrap")
	if err != nil {
		return "", ErrNoBwrap
	}
	return p, nil
}

// Spec is what the sandboxed game may reach beyond the read-only system.
type Spec struct {
	// Home is the user's home folder, which the sandbox replaces with an empty one.
	Home string
	// Runtime is XDG_RUNTIME_DIR; only the display, audio and input sockets named below are carried into it.
	Runtime string
	// RW are folders the game writes. Each must exist: a missing one fails the launch instead of letting the game
	// write into the throwaway home and lose its saves.
	RW []string
	// RO are folders the game reads; one that is missing is skipped.
	RO []string
	// Sockets are files under Runtime or elsewhere (Wayland, PipeWire, Pulse) that stay reachable; missing ones are skipped.
	Sockets []string
	// CacheDir holds the shader caches, so they persist without sharing the player's own.
	CacheDir string
	// Warn, when set, is shown in the run's console: something the game may write is not bound.
	Warn string
	// Bus is the host socket of a filtered session-bus proxy (StartProxy). Empty means the game gets no session bus.
	Bus string
	// Environ is the environment the game inherits; credential-looking variables are removed from it.
	Environ []string
}

// hidden are the places user files live. Each is emptied when it exists and is not a symlink.
var hidden = []string{"/home", "/root", "/mnt", "/media", "/run/media"}

// Argv is the bwrap command line that runs name with args. Binds apply in order, so a folder bound after the tmpfs
// that hides its parent shows through it. The host's PID and network namespaces are kept: Mortar finds and stops the
// game by process, and mods, multiplayer and the bridge need the network.
func Argv(bwrap string, s Spec, name string, args []string) ([]string, error) {
	if slices.Contains(s.RW, "") {
		return nil, errors.New("the sandbox was given an empty writable folder")
	}
	a := []string{
		bwrap, "--die-with-parent", "--new-session",
		"--ro-bind", "/", "/", "--dev-bind", "/dev", "/dev", "--proc", "/proc", "--tmpfs", "/tmp",
	}
	// The host's TMPDIR may sit outside the sandbox's writable folders; .NET fails to start when it cannot create its
	// diagnostics socket and debug pipes there, which looks like a crash.
	a = append(a, "--setenv", "TMPDIR", "/tmp")
	if exists("/tmp/.X11-unix") {
		a = append(a, "--bind", "/tmp/.X11-unix", "/tmp/.X11-unix")
	}
	for _, d := range append(slices.Clip(hidden), s.Home) {
		if d != "" && isDir(d) {
			a = append(a, "--tmpfs", d)
		}
	}
	if s.Runtime != "" && isDir(s.Runtime) {
		a = append(a, "--tmpfs", s.Runtime)
	}
	for _, p := range s.RO {
		if exists(p) {
			a = append(a, "--ro-bind", p, p)
		}
	}
	for _, p := range s.Sockets {
		if exists(p) {
			a = append(a, "--bind", p, p)
		}
	}
	for _, p := range s.RW {
		a = append(a, "--bind", p, p)
	}
	a = append(a, busArgs(s)...)
	if s.CacheDir != "" {
		a = append(a, "--bind", s.CacheDir, s.CacheDir,
			"--setenv", "MESA_SHADER_CACHE_DIR", s.CacheDir,
			"--setenv", "DXVK_STATE_CACHE_PATH", s.CacheDir,
			"--setenv", "__GL_SHADER_DISK_CACHE_PATH", s.CacheDir)
	}
	for _, k := range Credentials(s.Environ) {
		a = append(a, "--unsetenv", k)
	}
	return append(append(a, "--", name), args...), nil
}

// credentialMarks are fragments of variable names that carry a secret or a route to one.
var credentialMarks = []string{"TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "API_KEY", "AUTH_SOCK", "GPG_AGENT_INFO"}

// Credentials are the names in environ a game must not inherit.
func Credentials(environ []string) []string {
	var out []string
	for _, kv := range environ {
		k, _, _ := strings.Cut(kv, "=")
		up := strings.ToUpper(k)
		for _, m := range credentialMarks {
			if strings.Contains(up, m) {
				out = append(out, k)
				break
			}
		}
	}
	return out
}

// SteamRO are the parts of a Steam root the runtime and Proton read. The root's config, login tokens and userdata
// stay out, and so do other games' compatdata folders.
func SteamRO(root string) []string {
	var out []string
	for _, d := range []string{"ubuntu12_32", "ubuntu12_64", "linux32", "linux64", "compatibilitytools.d", "steamapps/common", "steam.sh"} {
		out = append(out, filepath.Join(root, d))
	}
	return out
}

// exists is false for "": safefs.Stat resolves it to a real entry, and a bind with an empty source fails the launch.
func exists(p string) bool {
	if p == "" {
		return false
	}
	_, err := safefs.Stat(p)
	return err == nil
}

func isDir(p string) bool {
	fi, err := os.Lstat(p)
	return err == nil && fi.IsDir()
}

// busArgs always sets DBUS_SESSION_BUS_ADDRESS: the host's address is useless in here (the runtime dir is emptied),
// and leaving it unset while DISPLAY is set makes libdbus autolaunch `dbus-launch`. The system bus socket is
// readable through the read-only root and its services (logind, polkit-gated units, NetworkManager) are not
// something a game needs, so /run/dbus is emptied and the address disabled.
func busArgs(s Spec) []string {
	var a []string
	addr := busOff
	if s.Bus != "" && s.Runtime != "" && isDir(s.Runtime) {
		in := filepath.Join(s.Runtime, "bus")
		a = append(a, "--bind", s.Bus, in)
		addr = "unix:path=" + in
	}
	if isDir("/run/dbus") {
		a = append(a, "--tmpfs", "/run/dbus")
	}
	return append(a, "--setenv", "DBUS_SESSION_BUS_ADDRESS", addr, "--setenv", "DBUS_SYSTEM_BUS_ADDRESS", busOff)
}
