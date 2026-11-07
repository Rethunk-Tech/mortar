package sandbox

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// ExtraFolders validates a path list of folders the player lets a sandboxed game write: each must be absolute, an
// existing folder, and neither the filesystem root nor the home folder itself, which would undo the sandbox.
func ExtraFolders(list, home string) ([]string, error) {
	var out []string
	for _, raw := range filepath.SplitList(list) {
		if raw == "" {
			continue
		}
		p := filepath.Clean(raw)
		switch {
		case !filepath.IsAbs(p):
			return nil, fmt.Errorf("the sandbox folder %q is not an absolute path", raw)
		case p == "/" || p == filepath.Clean(home):
			return nil, fmt.Errorf("the sandbox folder %q would expose everything the sandbox hides", raw)
		case !isDir(p):
			return nil, fmt.Errorf("the sandbox folder %q does not exist", raw)
		}
		out = append(out, p)
	}
	return out, nil
}

// Allowable is the folder to offer for a path a sandboxed run was refused: the path itself when it is a folder, else
// the nearest existing folder above it. Empty when that is the home folder, a place the sandbox empties as a whole
// (/home, /mnt) or the root, since allowing those would end the sandbox rather than fix one mod.
func Allowable(path, home string) string {
	p := filepath.Clean(path)
	for !isDir(p) {
		p = filepath.Dir(p)
	}
	if p == filepath.Clean(home) || p == "/" || !strings.HasPrefix(p, "/") {
		return ""
	}
	if slices.Contains(hidden, p) {
		return ""
	}
	return p
}
