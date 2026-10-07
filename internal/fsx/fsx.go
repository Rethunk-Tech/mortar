// Package fsx opens files through an os.Root on their parent directory, so a final path element
// that is a symlink cannot lead out of that directory.
package fsx

import (
	"crypto/md5" // #nosec G501 -- Nexus file hashes are MD5
	"crypto/sha256"
	"encoding/hex"
	"io"
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

// SHA256 is the hex SHA-256 of path's contents.
func SHA256(path string) (string, error) {
	f, err := Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// MD5 is the hex MD5 of path's contents; Nexus publishes its file hashes as MD5, so this is for comparing with them
// and nothing else.
func MD5(path string) (string, error) {
	f, err := Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := md5.New() // #nosec G401 -- Nexus file hashes are MD5
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

// IsDir is true when path exists and is a directory.
func IsDir(path string) bool {
	st, err := Stat(path)
	return err == nil && st.IsDir()
}
