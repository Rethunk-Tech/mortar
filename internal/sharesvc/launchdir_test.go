package sharesvc

import "testing"

func TestLaunchDirIsOWDOnlyInsideAnAppImage(t *testing.T) {
	t.Setenv("OWD", "/home/u/sub")
	t.Setenv("APPDIR", "")
	if got := LaunchDir(); got != "" {
		t.Fatalf("outside an AppImage LaunchDir = %q, want empty", got)
	}
	t.Setenv("APPDIR", "/tmp/.mount_mortar")
	if got := LaunchDir(); got != "/home/u/sub" {
		t.Fatalf("inside an AppImage LaunchDir = %q", got)
	}
	if got := InDir([]string{"t.mortar"}, LaunchDir()); got[0] != "/home/u/sub/t.mortar" {
		t.Fatalf("relative argument = %q", got[0])
	}
}
