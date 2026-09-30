// Package fsx opens files through an os.Root on their parent directory, so a final path element
// that is a symlink cannot lead out of that directory.
package fsx

import (
	"os"
	"path/filepath"
)

// in runs fn with a root on path's parent directory and path's final element.
func in[T any](path string, fn func(root *os.Root, name string) (T, error)) (T, error) {
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

// Create creates or truncates path.
func Create(path string) (*os.File, error) {
	return in(path, func(r *os.Root, name string) (*os.File, error) { return r.Create(name) })
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
