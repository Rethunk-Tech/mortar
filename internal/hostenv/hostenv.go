// Package hostenv gives child processes the environment Mortar was started from, minus what an AppImage's AppRun
// injected for Mortar's own bundled libraries.
package hostenv

import (
	"os"
	"path"
	"runtime"
	"strings"
)

// Environ is os.Environ for a child process. Inside an AppImage ($APPIMAGE and $APPDIR set) it drops every
// path-list entry under $APPDIR and every variable left empty by that, plus APPDIR, APPIMAGE and OWD, so a host tool
// such as git or dotnet loads the host's libraries instead of the bundle's. Mortar's own process keeps its
// environment: WebKit's helper processes need it. AppImages exist only on Linux, so elsewhere it is os.Environ.
func Environ() []string {
	if runtime.GOOS != "linux" {
		return os.Environ()
	}
	return scrub(os.Environ(), os.Getenv("APPIMAGE"), os.Getenv("APPDIR"))
}

func scrub(env []string, appimage, appdir string) []string {
	if appimage == "" || appdir == "" {
		return env
	}
	// An AppImage's paths are POSIX paths, whatever OS the tests run on.
	root := path.Clean(appdir)
	out := make([]string, 0, len(env))
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		switch k {
		case "APPDIR", "APPIMAGE", "OWD":
			continue
		}
		if !strings.Contains(v, root) {
			out = append(out, kv)
			continue
		}
		var keep []string
		for p := range strings.SplitSeq(v, ":") {
			if p != root && !strings.HasPrefix(path.Clean(p), root+"/") {
				keep = append(keep, p)
			}
		}
		if len(keep) > 0 {
			out = append(out, k+"="+strings.Join(keep, ":"))
		}
	}
	return out
}
