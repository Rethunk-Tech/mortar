package datadir

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// CopyTree copies the regular files and folders under src into dst (which may exist), file by file with io.Copy, which
// the OS clones where the filesystem can. Directory junctions and symlink directories are not followed. Symlink files
// are copied by content when they still resolve under src, and are an error when they escape.
type CopyProgress struct {
	Files      int
	TotalFiles int
	Bytes      int64
	TotalBytes int64
}

func CopyTree(src, dst string) error {
	return copyTree(src, dst, nil)
}

func copyTree(src, dst string, report func(CopyProgress)) error {
	root, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	var progress CopyProgress
	if report != nil {
		progress.TotalBytes, err = Size(src)
		if err != nil {
			return err
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
			return err
		}
	}
	return filepath.WalkDir(src, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
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
			st, err := os.Stat(resolved)
			if err != nil {
				return err
			}
			if st.IsDir() {
				return nil
			}
			if !UnderRoot(root, resolved) {
				return fmt.Errorf("%s escapes %s", p, src)
			}
			if err := CopyFile(resolved, target); err != nil {
				return err
			}
			if report != nil {
				progress.Files++
				info, err := os.Stat(resolved)
				if err != nil {
					return err
				}
				progress.Bytes += info.Size()
				report(progress)
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", p)
		}
		if !UnderRoot(root, resolved) {
			return fmt.Errorf("%s escapes %s", p, src)
		}
		if err := CopyFile(p, target); err != nil {
			return err
		}
		if report != nil {
			progress.Files++
			progress.Bytes += info.Size()
			report(progress)
		}
		return nil
	})
}

// UnderRoot reports whether p is root or a path still inside it after both have been cleaned.
func UnderRoot(root, p string) bool {
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	if root == p {
		return true
	}
	return strings.HasPrefix(p, root+string(os.PathSeparator))
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
	out, err := fsx.CreateExcl(dst, 0o600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	_, err = io.Copy(out, in)
	return err
}
