package fsx

import (
	"path/filepath"
	"strings"
)

// EvalSymlinks is filepath.EvalSymlinks that also works on paths past MAX_PATH, which the store's blobs reach.
func EvalSymlinks(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.EvalSymlinks(p)
	}
	extended := extendedPath(abs)
	resolved, err := filepath.EvalSymlinks(extended)
	if extended == abs {
		return resolved, err
	}
	return strings.TrimPrefix(resolved, `\\?\`), err
}
