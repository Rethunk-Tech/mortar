package launch

import (
	"cmp"
	"os"
	"path/filepath"
	"strings"
)

// systemDirs hold the operating system's own programs. No game runs from them, and a game whose process name is
// also a system program's (SUPERHOT's "SH" and /bin/sh) must never have that program stopped as the game.
var systemDirs = []string{"/bin", "/sbin", "/usr/bin", "/usr/sbin", "/usr/libexec", "/lib", "/lib64"}

// systemProgram reports whether path is a program in one of the system's own folders: systemDirs on Unix, the
// Windows folder on Windows.
func systemProgram(path string) bool {
	if path == "" {
		return false
	}
	if win := strings.ReplaceAll(path, `\`, "/"); len(win) > 2 && win[1] == ':' {
		root := strings.ReplaceAll(cmp.Or(os.Getenv("SystemRoot"), `C:\Windows`), `\`, "/")
		return strings.HasPrefix(strings.ToLower(win), strings.ToLower(strings.TrimRight(root, "/"))+"/")
	}
	dir := filepath.Dir(filepath.Clean(path))
	for _, d := range systemDirs {
		if dir == d || strings.HasPrefix(dir, d+"/") {
			return true
		}
	}
	return false
}
