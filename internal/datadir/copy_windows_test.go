//go:build windows

package datadir

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestCopyTreeSkipsAJunction(t *testing.T) {
	src, outside := t.TempDir(), t.TempDir()
	if err := fsx.WriteFile(filepath.Join(outside, "secret"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	junc := filepath.Join(src, "junc")
	if out, err := exec.CommandContext(t.Context(), "cmd", "/c", "mklink", "/J", junc, outside).CombinedOutput(); err != nil {
		t.Skipf("junction not permitted: %v %s", err, out)
	}
	dst := t.TempDir()
	skipped, err := copyTree(src, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 1 || skipped[0] != junc {
		t.Fatalf("skipped = %v", skipped)
	}
	if _, err := os.Stat(filepath.Join(dst, "a.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "junc")); err == nil {
		t.Fatal("junction was copied")
	}
}
