// Package fsx opens files through an os.Root on their parent directory, so a final path element
// that is a symlink cannot lead out of that directory.
package fsx

import (
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"path/filepath"
)

// in runs fn with a root on path's parent directory and path's final element. A trailing separator is dropped
// first: Dir keeps it as the last element while Base strips it, which would name that element twice.
func in[T any](path string, fn func(root *os.Root, name string) (T, error)) (T, error) {
	path = filepath.Clean(path)
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		var zero T
		return zero, err
	}
	defer func() { _ = root.Close() }()
	return fn(root, filepath.Base(path))
}

// Open opens path for reading.
func Open(path string) (*os.File, error) {
	return in(path, func(r *os.Root, name string) (*os.File, error) { return r.Open(name) })
}

// ErrNotRegular is OpenRegular's refusal of a directory, device, pipe or other non-regular file.
var ErrNotRegular = errors.New("not a regular file")

// OpenRegular opens path for reading, refusing anything but a regular file.
func OpenRegular(path string) (*os.File, error) {
	f, err := Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = ErrNotRegular
	}
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

// Create creates or truncates path.
func Create(path string) (*os.File, error) {
	return in(path, func(r *os.Root, name string) (*os.File, error) { return r.Create(name) })
}

// OpenFile opens path with flag and perm, still through the parent root.
func OpenFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	return in(path, func(r *os.Root, name string) (*os.File, error) {
		return r.OpenFile(name, flag, perm)
	})
}

// CreateExcl creates path with perm and fails if it already exists.
func CreateExcl(path string, perm os.FileMode) (*os.File, error) {
	return in(path, func(r *os.Root, name string) (*os.File, error) {
		return r.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	})
}

// Chmod sets path's permission bits.
func Chmod(path string, perm os.FileMode) error {
	_, err := in(path, func(r *os.Root, name string) (struct{}, error) {
		return struct{}{}, r.Chmod(name, perm)
	})
	return err
}

// ReadFile reads all of path.
func ReadFile(path string) ([]byte, error) {
	return in(path, func(r *os.Root, name string) ([]byte, error) { return r.ReadFile(name) })
}

// WriteFile writes data to path, creating it with perm if needed.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	_, err := in(path, func(r *os.Root, name string) (struct{}, error) {
		return struct{}{}, r.WriteFile(name, data, perm)
	})
	return err
}

// HashFile is the hex digest of path's contents under h.
func HashFile(path string, h hash.Hash) (string, error) {
	f, err := Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// IsFile is true when path resolves to a regular file. Unlike IsDir it uses os.Stat, so a symlink whose target
// is outside path's folder (a game exe linked from another library) still counts.
func IsFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

// Stat describes path.
func Stat(path string) (os.FileInfo, error) {
	return in(path, func(r *os.Root, name string) (os.FileInfo, error) { return r.Stat(name) })
}

// IsDir is true when path resolves to a directory. Like IsFile it follows a final symlink out of path's folder, so a
// symlinked game, Steam or library folder still counts.
func IsDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
