package nxm

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// onHost runs a command where the user's desktop lives: directly, or from inside a Flatpak through flatpak-spawn,
// where name is looked up on the host's PATH.
func (l *System) onHost(name string, args ...string) (string, error) {
	if inFlatpak() {
		hostName, hostArgs := sandbox.HostArgv("", nil, filepath.Base(name), args...)
		return l.run(hostName, hostArgs...)
	}
	return l.run(name, args...)
}

// launchOnHost starts the desktop entry id on link with the host's gio, finding the entry where the host's XDG
// data folders put it; the sandbox sees neither those folders nor the program.
const launchOnHost = `dirs="${XDG_DATA_HOME:-$HOME/.local/share}:${XDG_DATA_DIRS:-/usr/local/share:/usr/share}"
IFS=:
for d in $dirs; do
	if [ -f "$d/applications/$1" ]; then exec gio launch "$d/applications/$1" "$2"; fi
done
echo "no desktop entry $1" >&2
exit 1`

// The host's files below are the user's desktop configuration (mimeapps.list, browser manifests), which a Flatpak
// is not granted; inside one each step is a host sh script, else the plain file call.

// errHostMissing is the exit status hostRead's script gives a missing file.
const errHostMissing = 3

// hostConfigHome is the host's XDG config folder; a Flatpak's own XDG_CONFIG_HOME points into its sandbox.
func (l *System) hostConfigHome() (string, error) {
	if !inFlatpak() {
		return l.configHome, nil
	}
	return l.onHost("sh", "-c", `printf %s "${XDG_CONFIG_HOME:-$HOME/.config}"`)
}

// hostResolve follows a symlink (a dotfile manager's), so a rewrite lands in its target and the link stays.
func (l *System) hostResolve(path string) string {
	if inFlatpak() {
		if out, err := l.onHost("sh", "-c", `readlink -f "$1"`, "sh", path); err == nil && out != "" {
			return strings.TrimSuffix(out, "\n")
		}
		return path
	}
	if target, err := filepath.EvalSymlinks(path); err == nil {
		return target
	}
	return path
}

func (l *System) hostRead(path string) ([]byte, error) {
	if !inFlatpak() {
		return fsx.ReadFile(path)
	}
	out, err := l.onHost("sh", "-c", `[ -e "$1" ] || exit 3; cat "$1"`, "sh", path)
	if exit := (*exec.ExitError)(nil); errors.As(err, &exit) && exit.ExitCode() == errHostMissing {
		return nil, fs.ErrNotExist
	}
	return []byte(out), err
}

func (l *System) hostWrite(path string, b []byte) error {
	if !inFlatpak() {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		return datadir.WriteFile(path, b, desktopPerm)
	}
	_, err := l.onHost("sh", "-c", `mkdir -p "$(dirname "$1")" && printf %s "$2" >"$1.mortar-new" && mv "$1.mortar-new" "$1"`,
		"sh", path, string(b))
	return err
}

func (l *System) hostRemove(path string) error {
	if !inFlatpak() {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	_, err := l.onHost("sh", "-c", `rm -f "$1"`, "sh", path)
	return err
}

func (l *System) hostIsDir(path string) bool {
	if !inFlatpak() {
		return fsx.IsDir(path)
	}
	_, err := l.onHost("sh", "-c", `[ -d "$1" ]`, "sh", path)
	return err == nil
}
