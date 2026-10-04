package selfexe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchablePrefersImageOverMount(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "Mortar.AppImage")
	if err := os.WriteFile(img, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	mount := filepath.Join(dir, "mnt")
	t.Setenv("APPIMAGE", img)
	t.Setenv("APPDIR", mount)
	if got := Launchable(filepath.Join(mount, "usr", "bin", "mortar")); got != img {
		t.Fatalf("got %q, want the image %q", got, img)
	}
	if got := Launchable("/usr/bin/mortar"); got != "/usr/bin/mortar" {
		t.Fatalf("outside the mount must be unchanged, got %q", got)
	}
}
