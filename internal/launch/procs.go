package launch

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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

// LinkedFrom reports whether p's executable is the target of the file of the same name in dir. A mod manager that
// deploys by symlink (Vortex) puts the loader in the game folder as a link into its staging folder, and Windows names
// a process by the link's target, so the process path alone would not place it in the install.
func (p Process) LinkedFrom(dir string) bool {
	if p.Exe == "" || dir == "" {
		return false
	}
	target, err := fsx.EvalSymlinks(filepath.Join(dir, filepath.Base(p.Exe)))
	if err != nil {
		return false
	}
	return fsx.SamePath(target, p.Exe)
}

// ExeIs reports whether p's executable is one of the programs names, so its path is where the game itself runs
// from; it is false for an unreadable executable and for a host (Wine, Proton, dotnet) that runs the game.
func (p Process) ExeIs(names ...string) bool {
	if p.Exe == "" {
		return false
	}
	base := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(p.Exe), " (deleted)"), ".exe")
	for _, name := range names {
		if strings.EqualFold(base, name) {
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
	path, dir = fsx.FoldCase(filepath.Clean(path)), fsx.FoldCase(filepath.Clean(dir))
	return path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))
}

// UsesModsPath reports whether p was started with --mods-path pointing at modsDir. It is false when the
// platform gave no command line: see Process.Args.
func (p Process) UsesModsPath(modsDir string) bool {
	want := filepath.Clean(modsDir)
	for i, a := range p.Args {
		if v, ok := strings.CutPrefix(a, "--mods-path="); ok && fsx.SamePath(v, want) {
			return true
		}
		if a == "--mods-path" && i+1 < len(p.Args) && fsx.SamePath(p.Args[i+1], want) {
			return true
		}
	}
	return false
}
