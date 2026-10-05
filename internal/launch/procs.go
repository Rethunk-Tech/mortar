package launch

import (
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Process is a running process. Args is nil where the platform cannot give a command line (Windows); Exe is empty
// where the platform will not say which file the process runs (another user's process, a sandboxed host listing).
type Process struct {
	PID   int
	Exe   string
	Args  []string
	Start time.Time
}

// RunsFrom reports whether p belongs to the install in dir: its executable is inside dir, or a command-line
// argument names a path inside it. Wine and Proton processes show the Windows path of the game, which sits under
// Z: for a file outside the prefix, so that form counts as the Unix path it maps to.
func (p Process) RunsFrom(dir string) bool {
	if p.Exe != "" && within(p.Exe, dir) {
		return true
	}
	for _, a := range p.Args {
		if within(a, dir) || within(wineHostPath(a), dir) {
			return true
		}
	}
	return false
}

// wineHostPath maps a Wine Z: drive path to the Unix path it names, and returns anything else unchanged.
func wineHostPath(a string) string {
	if len(a) > 2 && (a[0] == 'Z' || a[0] == 'z') && a[1] == ':' {
		return strings.ReplaceAll(a[2:], `\`, "/")
	}
	return a
}

func within(path, dir string) bool {
	if path == "" || dir == "" {
		return false
	}
	path, dir = filepath.Clean(path), filepath.Clean(dir)
	if runtime.GOOS == "windows" {
		path, dir = strings.ToLower(path), strings.ToLower(dir)
	}
	return path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))
}

// UsesModsPath reports whether p was started with --mods-path pointing at modsDir. It is false when the
// platform gave no command line: see Process.Args.
func (p Process) UsesModsPath(modsDir string) bool {
	want := filepath.Clean(modsDir)
	for i, a := range p.Args {
		if v, ok := strings.CutPrefix(a, "--mods-path="); ok && filepath.Clean(v) == want {
			return true
		}
		if a == "--mods-path" && i+1 < len(p.Args) && filepath.Clean(p.Args[i+1]) == want {
			return true
		}
	}
	return false
}
