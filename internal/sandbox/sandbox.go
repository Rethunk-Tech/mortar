// Package sandbox builds the bubblewrap command line that starts a direct game launch with the user's home hidden, so
// code in a mod can reach only the folders the game needs.
package sandbox

import (
	"context"
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

// OverlaySupported reports whether the bwrap at path mounts temporary overlays (bubblewrap 0.9 or later).
func OverlaySupported(ctx context.Context, bwrap string) bool {
	out, err := exec.CommandContext(ctx, bwrap, "--help").CombinedOutput()
	return err == nil && strings.Contains(string(out), "--tmp-overlay")
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
	// Scratch are folders the game reads and must be able to write without the writes landing: Proton takes a lock
	// and unpacks into its own folder. Each gets an overlay whose writes go to a tmpfs, so a mod cannot change the
	// Proton every other game uses. Without overlay support in bwrap they are bound read-only, as RO is.
	Scratch []string
	// Overlay reports whether bwrap can mount the Scratch overlays (bubblewrap 0.9 or later).
	Overlay bool
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
	var links []string
	for _, p := range s.RO {
		if src, ok := source(p, s.Home); ok {
			dest, link := s.mountPoint(p, src)
			a, links = append(a, "--ro-bind", src, dest), append(links, link...)
		}
	}
	for _, p := range s.Scratch {
		src, ok := source(p, s.Home)
		if !ok || !isDir(src) {
			continue
		}
		dest, link := s.mountPoint(p, src)
		links = append(links, link...)
		if s.Overlay {
			a = append(a, "--overlay-src", src, "--tmp-overlay", dest)
		} else {
			a = append(a, "--ro-bind", src, dest)
		}
	}
	a = append(a, links...)
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

// source is the host path to mount at p: p itself, or what its links resolve to. A link into a folder the sandbox
// empties (Proton or a Steam runtime linked in from the real home or another drive) would otherwise point at nothing,
// and a link may not lead into the player's keys and sign-ins. bwrap reads a bind's source from the host, so the
// target need not be visible inside.
func source(p, home string) (string, bool) {
	if p == "" {
		return "", false
	}
	target, err := filepath.EvalSymlinks(p)
	if err != nil || !exists(target) || (home != "" && holdsSecrets(target, home)) {
		return "", false
	}
	return target, true
}

// mountPoint is where src is mounted for the folder p. bwrap cannot mount onto a symlink, so a linked p is mounted at
// its target, and the link is made again inside when p sits in a folder the sandbox empties.
func (s Spec) mountPoint(p, src string) (string, []string) {
	if fi, err := os.Lstat(p); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return p, nil
	}
	if !s.hides(p) {
		return src, nil
	}
	return src, []string{"--dir", filepath.Dir(p), "--symlink", src, p}
}

// hides reports whether p is inside a folder the sandbox replaces with an empty one.
func (s Spec) hides(p string) bool {
	for _, d := range append(slices.Clip(hidden), s.Home, s.Runtime) {
		if d != "" && (p == d || strings.HasPrefix(p, strings.TrimSuffix(d, "/")+"/")) {
			return true
		}
	}
	return false
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
