//go:build !windows

package shortcut

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestRenamedDesktopEntryWithoutNameKeepsMortarWording(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path, err := desktopPath("stardew", "p1")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(path, []byte("[Desktop Entry]\nType=Application\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = Renamed("stardew", "p1", "Renamed", "Stardew Valley")
	if err == nil || err.Error() != "desktop entry "+path+" has no Name" {
		t.Fatalf("got %v", err)
	}
}
