package launch

import (
	"path/filepath"
	"strings"
	"time"
)

// Process is a running process. Args is nil where the platform cannot give a command line (Windows).
type Process struct {
	PID   int
	Args  []string
	Start time.Time
}

// runs reports whether args start the program name, directly or through dotnet or mono.
func runs(args []string, name string) bool {
	is := func(a string) bool {
		base := filepath.Base(a)
		return strings.EqualFold(strings.TrimSuffix(strings.TrimSuffix(base, ".exe"), ".dll"), name)
	}
	if len(args) == 0 {
		return false
	}
	if is(args[0]) {
		return true
	}
	host := filepath.Base(args[0])
	return (host == "dotnet" || host == "mono") && len(args) > 1 && is(args[1])
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
