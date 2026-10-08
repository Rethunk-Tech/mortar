package fsx

import (
	"io"
	"os"
	"path/filepath"
)

// Shared reports whether path is a symlink or a file with other hard links: writing it would change the file another
// folder (a mod manager's staging folder) holds too.
func Shared(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return true
	}
	n, err := LinkCount(path)
	return err == nil && fi.Mode().IsRegular() && n > 1
}

// Unshare replaces a shared file with a private copy of its content, leaving the other folder's file as it was.
func Unshare(path string) error {
	if !Shared(path) {
		return nil
	}
	src, err := os.Open(path) // #nosec G304 -- path is a game file the caller names
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	fi, err := src.Stat()
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mortar-unshare-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(fi.Mode().Perm() | 0o200); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = src.Close()
	return Rename(tmp.Name(), path)
}
