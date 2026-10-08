// Package appversion reads Mortar's version from build/config.yml, the one place it is set, and names the running build.
package appversion

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"sync"
)

// infoVersion matches `version: "x"` indented under `info:`; the file's top-level `version: '3'` is the
// Taskfile schema version, not the app's.
var infoVersion = regexp.MustCompile(`(?m)^info:\n(?:[ \t]+.*\n)*?[ \t]+version:[ \t]*"([^"]+)"`)

// FromConfig returns info.version from the contents of build/config.yml.
func FromConfig(config []byte) (string, error) {
	m := infoVersion.FindSubmatch(config)
	if m == nil {
		return "", errors.New("build/config.yml has no info.version")
	}
	return string(m[1]), nil
}

// Build names this exact build, so a cache another build wrote is known to be stale: the commit for a clean tree,
// with the executable's size and time added otherwise, since every local build shares the commit.
var Build = sync.OnceValue(func() string {
	rev, modified := "", true
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				modified = s.Value == "true"
			}
		}
	}
	if rev != "" && !modified {
		return rev
	}
	if exe, err := os.Executable(); err == nil {
		if fi, err := os.Stat(exe); err == nil {
			return fmt.Sprintf("%s+%d.%d", rev, fi.Size(), fi.ModTime().UnixNano())
		}
	}
	return rev
})

// Commit is the short commit this binary was built from, with "-modified" when the tree had uncommitted changes, so
// two builds of one version can be told apart; empty when the build carries no VCS stamp.
var Commit = sync.OnceValue(func() string {
	rev, modified := "", false
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				modified = s.Value == "true"
			}
		}
	}
	if len(rev) > 8 {
		rev = rev[:8]
	}
	if rev != "" && modified {
		rev += "-modified"
	}
	return rev
})
