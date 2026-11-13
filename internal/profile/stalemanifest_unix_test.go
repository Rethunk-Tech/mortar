//go:build unix

package profile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

// Not parallel: the umask is the whole process's.
func TestFixStaleManifestKeepsTheFilesModeUnderARestrictiveUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	t.Cleanup(func() { syscall.Umask(old) })
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := safefs.WriteFile(path, []byte(manifestJSON("X.A")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o664); err != nil {
		t.Fatal(err)
	}
	if err := setManifestVersion(path, "smapi:x.a", "1.2.0"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o664 {
		t.Fatalf("mode after the fix = %o, want 664", got)
	}
}
