package datadir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestCopyTreeCopiesSymlinkFilesThatStayInside(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "real.png"), []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(src, "assets"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(src, "real.png"), filepath.Join(src, "assets", "foo.png")); err != nil {
		t.Fatal(err)
	}
	if err := CopyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(filepath.Join(dst, "assets", "foo.png"))
	if err != nil || string(got) != "png" {
		t.Fatalf("copied symlink = %q, %v", got, err)
	}
}

func TestCopyTreeRejectsASymlinkFileThatEscapes(t *testing.T) {
	src := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	if err := fsx.WriteFile(outside, []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(src, "link")); err != nil {
		t.Fatal(err)
	}
	err := CopyTree(src, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("escape = %v", err)
	}
}

func TestCopyTreeSkipsASymlinkDirectory(t *testing.T) {
	src := t.TempDir()
	outside := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(outside, "secret.txt"), []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(src, "ok.txt"), []byte("yes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(src, "Innocent")); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := CopyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(dst, "Innocent")); !os.IsNotExist(err) {
		t.Fatalf("symlink dir was copied: %v", err)
	}
	got, err := fsx.ReadFile(filepath.Join(dst, "ok.txt"))
	if err != nil || string(got) != "yes" {
		t.Fatalf("regular file = %q, %v", got, err)
	}
}

func TestUnderRoot(t *testing.T) {
	if !UnderRoot("/", "/etc") {
		t.Fatal("path under filesystem root")
	}
	if !UnderRoot("/data", "/data") {
		t.Fatal("path equal to root")
	}
	if UnderRoot("/data", "/data-other") {
		t.Fatal("sibling prefix")
	}
	if UnderRoot("/data", "/etc") {
		t.Fatal("path outside root")
	}
}

func TestCopyTreeReportsSkippedLinkedFolders(t *testing.T) {
	src, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(src, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	skipped, err := copyTree(src, t.TempDir(), nil)
	if err != nil || len(skipped) != 1 {
		t.Fatalf("skipped = %v, err = %v", skipped, err)
	}
}
