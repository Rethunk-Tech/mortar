package fsx

import (
	"io/fs"
	"strings"
)

// ResolveFold resolves the slash path rel below fsys one element at a time, backtracking. At each element it tries the
// entry of exactly that name first, then each entry strings.EqualFold matches, in fs.ReadDir order (sorted by name),
// and returns the first whose remaining path resolves. It finds a file whenever some path in fsys matches rel element
// by element under case folding, which is how a path written for Windows reads on a case-sensitive file system.
//
// rel must pass fs.ValidPath ("." returns "."), else the error is a *fs.PathError wrapping fs.ErrInvalid. No match, or
// a walk past its ReadDir budget (8 per element of rel), is a *fs.PathError wrapping fs.ErrNotExist. In a tree each
// directory is read at most once; only a linked directory, reachable under several names, can make the walk long.
func ResolveFold(fsys fs.FS, rel string) (string, error) {
	if !fs.ValidPath(rel) {
		return "", &fs.PathError{Op: "resolvefold", Path: rel, Err: fs.ErrInvalid}
	}
	if rel == "." {
		return ".", nil
	}
	parts := strings.Split(rel, "/")
	budget := 8 * len(parts)
	var walk func(dir string, i int) (string, bool)
	walk = func(dir string, i int) (string, bool) {
		if budget == 0 {
			return "", false
		}
		budget--
		entries, err := fs.ReadDir(fsys, dir)
		if err != nil {
			return "", false
		}
		try := func(name string) (string, bool) {
			next := name
			if dir != "." {
				next = dir + "/" + name
			}
			if i == len(parts)-1 {
				return next, true
			}
			return walk(next, i+1)
		}
		for _, e := range entries {
			if e.Name() == parts[i] {
				if got, ok := try(e.Name()); ok {
					return got, true
				}
				break
			}
		}
		for _, e := range entries {
			if e.Name() != parts[i] && strings.EqualFold(e.Name(), parts[i]) {
				if got, ok := try(e.Name()); ok {
					return got, true
				}
			}
		}
		return "", false
	}
	if got, ok := walk(".", 0); ok {
		return got, nil
	}
	return "", &fs.PathError{Op: "resolvefold", Path: rel, Err: fs.ErrNotExist}
}
