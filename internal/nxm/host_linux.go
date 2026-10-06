package nxm

import (
	"path/filepath"

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
