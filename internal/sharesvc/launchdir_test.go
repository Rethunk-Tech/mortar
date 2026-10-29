package sharesvc

import (
	"path/filepath"
	"testing"
)

func TestLaunchDirIsOWDOnlyInsideAnAppImage(t *testing.T) {
	owd := filepath.Join(t.TempDir(), "sub")
	t.Setenv("OWD", owd)
	t.Setenv("APPDIR", "")
	if got := LaunchDir(); got != "" {
		t.Fatalf("outside an AppImage LaunchDir = %q, want empty", got)
	}
	t.Setenv("APPDIR", "/tmp/.mount_mortar")
	if got := LaunchDir(); got != owd {
		t.Fatalf("inside an AppImage LaunchDir = %q", got)
	}
	if got := InDir([]string{"t.mortar"}, LaunchDir()); got[0] != filepath.Join(owd, "t.mortar") {
		t.Fatalf("relative argument = %q", got[0])
	}
}
