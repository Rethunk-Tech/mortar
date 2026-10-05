package datadir

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// CopyProgress counts files and bytes while CopyTree runs.
type CopyProgress struct {
	Files      int
	TotalFiles int
	Bytes      int64
	TotalBytes int64
}

// CopyTree copies the regular files and folders under src into dst (which may exist), file by file with io.Copy.
// Directory junctions, mount points and symlink directories are not followed. Symlink files are copied by content
// when they still resolve under src, and are an error when they escape.
func CopyTree(src, dst string) error {
	_, err := copyTree(src, dst, nil)
	return err
}

// LinkedDir reports an entry that leads to a directory without being one: a symlink to a folder, or on Windows a
// junction or mount point, which Go reports as an irregular file.
func LinkedDir(p string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 && info.Mode().IsRegular() {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// copyTree is CopyTree with a custom per-file put; it also returns the linked folders it skipped.
func copyTree(src, dst string, put func(from, to, rel string) error) (skipped []string, err error) {
	return copyTreeFirst(src, dst, "", nil, put)
}

// copyTreeFirst is copyTree that, when first names a top-level entry of src, copies that entry before the rest.
func copyTreeFirst(src, dst, first string, report func(CopyProgress), put func(from, to, rel string) error) (skipped []string, err error) {
	if put == nil {
		put = func(from, to, _ string) error { return CopyFile(from, to) }
	}
	root, err := filepath.EvalSymlinks(src)
	if err != nil {
		return nil, err
	}
	var progress CopyProgress
	if report != nil {
		progress.TotalBytes, err = Size(src)
		if err != nil {
			return nil, err
		}
		err = filepath.WalkDir(src, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.Type().IsRegular() {
				progress.TotalFiles++
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	passes := 1
	if first != "" {
		passes = 2
	}
	for pass := 0; pass < passes && err == nil; pass++ {
		err = walkCopy(src, dst, first, pass, &progress, report, put, root, &skipped)
	}
	return skipped, err
}

// walkCopy is one walk of copyTreeFirst. In a two-pass copy, pass 0 takes only the first entry and pass 1 the rest.
func walkCopy(src, dst, first string, pass int, progress *CopyProgress, report func(CopyProgress), put func(from, to, rel string) error, root string, skipped *[]string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if first != "" && p != src {
			top, relErr := filepath.Rel(src, p)
			if relErr != nil {
				return relErr
			}
			if (strings.SplitN(filepath.ToSlash(top), "/", 2)[0] == first) != (pass == 0) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if p != src && !RealDirUnder(root, p) {
				return fs.SkipDir
			}
			rel, err := filepath.Rel(src, p)
			if err != nil {
				return err
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o750)
		}
		if LinkedDir(p, info) {
			*skipped = append(*skipped, p)
			return nil
		}
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			if !UnderRoot(root, resolved) {
				return fmt.Errorf("%s escapes %s", p, src)
			}
			if err := put(resolved, target, rel); err != nil {
				return err
			}
			if report != nil {
				progress.Files++
				info, err := os.Stat(resolved)
				if err != nil {
					return err
				}
				progress.Bytes += info.Size()
				report(*progress)
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", p)
		}
		if !UnderRoot(root, resolved) {
			return fmt.Errorf("%s escapes %s", p, src)
		}
		if err := put(p, target, rel); err != nil {
			return err
		}
		if report != nil {
			progress.Files++
			progress.Bytes += info.Size()
			report(*progress)
		}
		return nil
	})
}

// UnderRoot reports whether p is root or a path still inside it after both have been cleaned.
func UnderRoot(root, p string) bool {
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// RealDirUnder reports whether p is a directory whose own path still resolves under root, not a symlink
// directory or a junction/reparse that points somewhere else.
func RealDirUnder(root, p string) bool {
	info, err := os.Lstat(p)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return false
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(p))
	if err != nil {
		return false
	}
	if filepath.Clean(resolved) != filepath.Clean(filepath.Join(parent, filepath.Base(p))) {
		return false
	}
	return UnderRoot(root, resolved)
}

// CopyFile copies the regular file src to dst, which must not exist yet.
func CopyFile(src, dst string) (err error) {
	in, err := fsx.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	// The copy keeps the source's exec and read bits but never grants write to group or others.
	out, err := fsx.CreateExcl(dst, info.Mode().Perm()&0o755)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	_, err = io.Copy(out, in)
	return err
}
