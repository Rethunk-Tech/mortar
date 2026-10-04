package datadir

import (
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
var portable = sync.OnceValue(func() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return portableDir(exe)
})

// Portable reports whether this process keeps its data beside the executable.
func Portable() bool { return portable() != "" }

// portableDir is <exe dir>/data when the marker sits beside exe and that folder is writable, else "". Inside a
// Flatpak the executable lives in the read-only /app mount, and inside an AppImage in its read-only squashfs mount
// that vanishes on exit; both are ignored even if a marker were baked in.
func portableDir(exe string) string {
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	if os.Getenv("FLATPAK_ID") != "" || dir == "/app" || strings.HasPrefix(dir, "/app/") {
		return ""
	}
	if info, err := fsx.Stat(filepath.Join(dir, PortableMarker)); err != nil || !info.Mode().IsRegular() {
		return ""
	}
	probe, err := os.CreateTemp(dir, ".mortar-write-*")
	if err != nil {
		return ""
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	return filepath.Join(dir, "data")
}
