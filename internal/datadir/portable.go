package datadir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// PortableMarker is the file beside the executable that keeps the data folder in "data" next to it.
const PortableMarker = "portable"

// portable is decided once per process: a marker added or removed while Mortar runs takes effect on the next start,
// never by moving the data folder under a running session.
var portable = sync.OnceValues(func() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", nil
	}
	return portableDir(exe)
})

// Portable reports whether this process keeps its data beside the executable.
func Portable() bool {
	dir, _ := portable()
	return dir != ""
}

// PortableError says the portable marker is present but its folder cannot be written, so Mortar will not quietly keep
// the data on the host while the user believes it travels with the media.
type PortableError struct {
	Dir string
	Err error
}

func (e *PortableError) Error() string {
	return fmt.Sprintf("Mortar is set up as portable (a \"portable\" file sits beside it), but %s cannot be written to: %v. "+
		"Move Mortar to a writable folder, or delete the \"portable\" file to use the normal data folder.", e.Dir, e.Err)
}

func (e *PortableError) Unwrap() error { return e.Err }

// portableDir is <exe dir>/data when the marker sits beside exe and that folder is writable, "" without a marker, and
// a *PortableError with a marker in an unwritable folder. Inside a Flatpak the executable lives in the read-only /app
// mount, and inside an AppImage in its read-only squashfs mount that vanishes on exit; both are ignored even if a
// marker were baked in.
func portableDir(exe string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	if os.Getenv("FLATPAK_ID") != "" || dir == "/app" || strings.HasPrefix(dir, "/app/") {
		return "", nil
	}
	if info, err := fsx.Stat(filepath.Join(dir, PortableMarker)); err != nil || !info.Mode().IsRegular() {
		return "", nil
	}
	probe, err := os.CreateTemp(dir, ".mortar-write-*")
	if err != nil {
		return "", &PortableError{Dir: dir, Err: err}
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	return filepath.Join(dir, "data"), nil
}
