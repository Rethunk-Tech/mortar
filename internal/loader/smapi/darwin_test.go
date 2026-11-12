package smapi

import (
	"path/filepath"
	"testing"
)

func TestGameFolder(t *testing.T) {
	dir := filepath.Join("games", "Stardew Valley")
	if got, want := gameFolder(dir, "darwin"), filepath.Join(dir, "Contents", "MacOS"); got != want {
		t.Errorf("darwin: %q, want %q", got, want)
	}
	for _, goos := range []string{"linux", "windows"} {
		if got := gameFolder(dir, goos); got != dir {
			t.Errorf("%s: %q, want the install itself", goos, got)
		}
	}
	if d, exe, err := installer("1.2.3", "darwin"); err != nil || exe != "SMAPI.Installer" || filepath.Base(d) != "macOS" {
		t.Errorf("installer: %q %q %v", d, exe, err)
	}
}
