// Package testfs holds file, zip and data-home helpers for tests; it imports no mortar package above fsx, so any test package may use it.
package testfs

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// ZipBytes returns a zip holding files (name to body).
func ZipBytes(t testing.TB, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// WriteZip writes a zip of files to path and returns path.
func WriteZip(t testing.TB, path string, files map[string]string) string {
	t.Helper()
	if err := fsx.WriteFile(path, ZipBytes(t, files), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// WriteFile writes body at the slash-separated rel under root, creating folders, and returns the full path.
func WriteFile(t testing.TB, root, rel, body string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// DataHome points the per-user data folder at a fresh temp dir on every platform and returns it.
func DataHome(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	return dir
}
