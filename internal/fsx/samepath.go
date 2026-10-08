package fsx

import "path/filepath"

// SamePath reports whether a and b name the same location once cleaned and case-folded where the file system ignores
// case (FoldCase).
func SamePath(a, b string) bool {
	return FoldCase(filepath.Clean(a)) == FoldCase(filepath.Clean(b))
}
