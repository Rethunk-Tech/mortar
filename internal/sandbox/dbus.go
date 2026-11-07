package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// busOff is the address libdbus and GDBus both read as "no bus": unlike an unset variable it never makes them
// autolaunch `dbus-launch`, which inside the sandbox (no session bus socket, no_new_privs) forks without bound.
const busOff = "disabled:"

// busTalk are the session-bus names a game may call through the filtered proxy: GameMode, screensaver inhibit and
// the desktop portals. Everything else on the bus (units, keyring, other applications) stays unreachable.
var busTalk = []string{
	"com.feralinteractive.GameMode",
	"org.freedesktop.ScreenSaver",
	"org.freedesktop.portal.*",
}

// TasksMax caps the processes and threads of one sandboxed game. Proton games peak at a few hundred threads; this
// leaves several times that, and a runaway fork loop stops far short of the user's own limit.
const TasksMax = 4096

// ProxyArgv is the xdg-dbus-proxy command line that exposes busTalk of the bus at upstream on socket. The proxy
// writes a byte to fd once it listens and exits when the other end of fd closes.
func ProxyArgv(proxy, upstream, socket string, fd int) []string {
	a := []string{proxy, upstream, socket, "--filter", "--fd=" + strconv.Itoa(fd)}
	for _, n := range busTalk {
		a = append(a, "--talk="+n)
	}
	return a
}

// Bus is a running filtered session-bus proxy.
type Bus struct {
	// Socket is the proxy's listening socket on the host.
	Socket string
	hold   *os.File
	dir    string
	cmd    *exec.Cmd
}

// Close stops the proxy and removes its socket. It is safe on nil and to call twice.
func (b *Bus) Close() {
	if b == nil || b.hold == nil {
		return
	}
	_ = b.hold.Close()
	_ = b.cmd.Wait()
	_ = os.RemoveAll(b.dir)
	b.hold = nil
}

// SessionAddress is the host session bus address from environ, or the default socket under runtime.
func SessionAddress(environ []string, runtime string) string {
	for _, e := range slices.Backward(environ) {
		if v, ok := strings.CutPrefix(e, "DBUS_SESSION_BUS_ADDRESS="); ok && v != "" {
			return v
		}
	}
	if runtime == "" {
		return ""
	}
	return "unix:path=" + filepath.Join(runtime, "bus")
}

// ErrNoProxy means xdg-dbus-proxy is not installed.
var ErrNoProxy = errors.New("xdg-dbus-proxy is not installed")

// StartProxy runs a filtered proxy of the session bus at upstream, with its socket in a fresh folder under runtime.
// The caller closes it when the game ends; if Mortar dies first the held pipe closes and the proxy exits anyway.
func StartProxy(upstream, runtime string) (*Bus, error) {
	proxy, err := exec.LookPath("xdg-dbus-proxy")
	if err != nil {
		return nil, ErrNoProxy
	}
	if path, ok := strings.CutPrefix(upstream, "unix:path="); !ok || !exists(strings.SplitN(path, ",", 2)[0]) {
		return nil, errors.New("no session bus socket to proxy")
	}
	dir, err := os.MkdirTemp(runtime, "mortar-bus-")
	if err != nil {
		return nil, err
	}
	r, w, err := os.Pipe()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	sock := filepath.Join(dir, "bus")
	argv := ProxyArgv(proxy, upstream, sock, 3)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.ExtraFiles = []*os.File{w}
	err = cmd.Start()
	_ = w.Close()
	if err != nil {
		_ = r.Close()
		_ = os.RemoveAll(dir)
		return nil, err
	}
	b := &Bus{Socket: sock, hold: r, dir: dir, cmd: cmd}
	_ = r.SetReadDeadline(time.Now().Add(5 * time.Second))
	if n, err := r.Read(make([]byte, 1)); err != nil || n != 1 {
		b.Close()
		return nil, errors.New("xdg-dbus-proxy did not start")
	}
	return b, nil
}

// TaskCap is the prefix that runs a command in a transient systemd user scope limited to max tasks, or a note saying
// why the launch goes without one. systemd-run --scope execs the command in place, so Mortar still tracks its pid.
func TaskCap(max int) (prefix []string, note string) {
	p, err := exec.LookPath("systemd-run")
	if err != nil {
		return nil, "the process cap was not applied: systemd-run is not installed"
	}
	prefix = []string{p, "--user", "--scope", "--quiet", "-p", "TasksMax=" + strconv.Itoa(max), "--"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	probe := append(append([]string{}, prefix...), "true")
	if err := exec.CommandContext(ctx, probe[0], probe[1:]...).Run(); err != nil {
		return nil, "the process cap was not applied: no systemd user manager answered (" + err.Error() + ")"
	}
	return prefix, ""
}
