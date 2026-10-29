// Package testfs holds file, zip and data-home helpers for tests; it imports no mortar package above fsx, so any test package may use it.
package testfs

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	WriteAt(t, p, []byte(body))
	return p
}

// WriteAt writes body at path, creating its folder.
func WriteAt(t testing.TB, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Read is the file at path as a string.
func Read(t testing.TB, path string) string {
	t.Helper()
	b, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Touch appends a blank line to the file at path and dates it a minute back, past the window in which a fresh stamp is
// not trusted. The file's bytes and time come back when the test ends.
func Touch(t testing.TB, path string) {
	t.Helper()
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = fsx.WriteFile(path, raw, info.Mode())
		_ = os.Chtimes(path, info.ModTime(), info.ModTime())
	})
	if err := fsx.WriteFile(path, append(slices.Clone(raw), '\n'), info.Mode()); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(-time.Minute)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

// WriteFarm writes a Stardew Valley save folder under saves: its save file holding body, and SaveGameInfo naming the
// farm.
func WriteFarm(t testing.TB, saves, folder, farm, body string) {
	t.Helper()
	dir := filepath.Join(saves, folder)
	WriteFile(t, dir, folder, body)
	WriteFile(t, dir, "SaveGameInfo", `<Farmer><name>Ann</name><farmName>`+farm+`</farmName></Farmer>`)
}

// DataHome points the per-user data folder at a fresh temp dir on every platform and returns it.
func DataHome(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
	return dir
}
