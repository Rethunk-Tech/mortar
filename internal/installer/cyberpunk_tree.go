package installer

import (
	"path"
	"slices"
	"sort"
	"strings"
)

// ftree is an archive's file list as the Cyberpunk 2077 installer reads it: slash paths with no folders, folder names
// compared without regard to case, as Windows and Wine do. The query names follow Vortex's cyberpunk2077_ext_redux
// filetree.ts, which the layouts below are written against; dir "" is the archive's top.
type ftree struct{ files []string }

func newFtree(files []string) ftree {
	out := slices.Clone(files)
	sort.Strings(out)
	return ftree{files: out}
}

func lower(s string) string { return strings.ToLower(s) }

func isUnder(f, dir string) bool {
	return dir == "" || strings.HasPrefix(lower(f), lower(dir)+"/")
}

// filesIn are the files directly in dir that pred takes.
func (t ftree) filesIn(dir string, pred func(string) bool) []string {
	var out []string
	for _, f := range t.files {
		if isUnder(f, dir) && !strings.Contains(strings.TrimPrefix(f[len(dir):], "/"), "/") && (pred == nil || pred(f)) {
			out = append(out, f)
		}
	}
	return out
}

// filesUnder are the files in dir or below it that pred takes.
func (t ftree) filesUnder(dir string, pred func(string) bool) []string {
	var out []string
	for _, f := range t.files {
		if isUnder(f, dir) && (pred == nil || pred(f)) {
			out = append(out, f)
		}
	}
	return out
}

// subdirNamesIn are the names of the folders directly in dir.
func (t ftree) subdirNamesIn(dir string) []string {
	var out []string
	for _, f := range t.files {
		if !isUnder(f, dir) {
			continue
		}
		rest := strings.TrimPrefix(f[len(dir):], "/")
		if name, _, ok := strings.Cut(rest, "/"); ok && !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

func (t ftree) dirInTree(dir string) bool {
	return dir == "" || slices.ContainsFunc(t.files, func(f string) bool { return isUnder(f, dir) })
}

func (t ftree) pathInTree(p string) bool {
	return slices.ContainsFunc(t.files, func(f string) bool { return strings.EqualFold(f, p) })
}

// dirsWithSome walks the folders below dir that directly hold a file pred takes. topmost stops at the first such
// folder on each branch; direct looks only at dir's own subfolders.
func (t ftree) dirsWithSome(dir string, pred func(string) bool, topmost, direct bool) []string {
	var dirs []string
	for _, f := range t.files {
		if !isUnder(f, dir) || !pred(f) {
			continue
		}
		d := path.Dir(f)
		if d == "." || d == "" || strings.EqualFold(d, dir) {
			continue
		}
		if !slices.ContainsFunc(dirs, func(x string) bool { return strings.EqualFold(x, d) }) {
			dirs = append(dirs, d)
		}
	}
	if direct {
		dirs = slices.DeleteFunc(dirs, func(d string) bool { return !strings.EqualFold(path.Dir(d), orDot(dir)) })
	}
	if topmost {
		dirs = slices.DeleteFunc(dirs, func(d string) bool {
			return slices.ContainsFunc(dirs, func(o string) bool { return !strings.EqualFold(o, d) && isUnder(d, o) })
		})
	}
	sort.Strings(dirs)
	return dirs
}

func orDot(dir string) string {
	if dir == "" {
		return "."
	}
	return dir
}

func (t ftree) allSubdirsWithSome(dir string, pred func(string) bool) []string {
	return t.dirsWithSome(dir, pred, false, false)
}

func (t ftree) topmostSubdirsWithSome(dir string, pred func(string) bool) []string {
	return t.dirsWithSome(dir, pred, true, false)
}

func (t ftree) directSubdirsWithSome(dir string, pred func(string) bool) []string {
	return t.dirsWithSome(dir, pred, false, true)
}

func (t ftree) dirWithSomeIn(dir string, pred func(string) bool) bool {
	return len(t.filesIn(dir, pred)) > 0
}

func (t ftree) dirWithSomeUnder(dir string, pred func(string) bool) bool {
	return len(t.filesUnder(dir, pred)) > 0
}

func extIs(exts ...string) func(string) bool {
	return func(f string) bool { return slices.Contains(exts, lower(path.Ext(f))) }
}

func base(f string) string { return path.Base(f) }

// replacePrefix swaps from for to at the start of f, whatever case f's folder names have.
func replacePrefix(f, from, to string) string {
	if from == "" {
		return path.Join(to, f)
	}
	if len(f) >= len(from) && strings.EqualFold(f[:len(from)], from) {
		return path.Join(to, f[len(from):])
	}
	return f
}
