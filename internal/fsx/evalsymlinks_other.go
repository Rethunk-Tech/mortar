//go:build !windows

package fsx

import "path/filepath"

// EvalSymlinks is filepath.EvalSymlinks; only Windows needs help with long paths.
func EvalSymlinks(p string) (string, error) { return filepath.EvalSymlinks(p) }
