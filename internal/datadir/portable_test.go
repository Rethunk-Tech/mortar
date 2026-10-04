package datadir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func fakeExe(t *testing.T, marker bool) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "mortar")
	if err := fsx.WriteFile(exe, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	if marker {
		if err := fsx.WriteFile(filepath.Join(dir, PortableMarker), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return exe
}

func TestPortableDirNeedsTheMarkerBesideTheResolvedExecutable(t *testing.T) {
	t.Setenv("FLATPAK_ID", "")
	if got := portableDir(fakeExe(t, false)); got != "" {
		t.Fatalf("no marker = %q", got)
	}
	exe := fakeExe(t, true)
	want := filepath.Join(filepath.Dir(exe), "data")
	if got := portableDir(exe); got != want {
		t.Fatalf("marker = %q, want %q", got, want)
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(t.TempDir(), "mortar")
		if err := os.Symlink(exe, link); err != nil {
			t.Fatal(err)
		}
		if got := portableDir(link); got != want {
			t.Fatalf("through a symlink = %q, want %q", got, want)
		}
	}
	t.Setenv("FLATPAK_ID", "tech.rethunk.Mortar")
	if got := portableDir(exe); got != "" {
		t.Fatalf("inside a Flatpak = %q", got)
	}
}

func TestPortableDirIgnoresAReadOnlyFolder(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions enforced")
	}
	t.Setenv("FLATPAK_ID", "")
	exe := fakeExe(t, true)
	dir := filepath.Dir(exe)
	if err := fsx.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fsx.Chmod(dir, 0o700) })
	if got := portableDir(exe); got != "" {
		t.Fatalf("read-only = %q", got)
	}
}

func TestPortableModeIgnoresThePointer(t *testing.T) {
	def := t.TempDir()
	t.Setenv("XDG_DATA_HOME", def)
	t.Setenv("LOCALAPPDATA", def)
	d, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(d, PointerName), []byte(t.TempDir()), 0o600); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(t.TempDir(), "data")
	old := portable
	portable = func() string { return want }
	t.Cleanup(func() { portable = old })
	got, err := Dir()
	if err != nil || got != want {
		t.Fatalf("Dir = %q %v, want %q", got, err, want)
	}
	if !Portable() {
		t.Fatal("Portable() = false")
	}
}
