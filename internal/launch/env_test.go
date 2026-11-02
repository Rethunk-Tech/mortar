package launch

import (
	"slices"
	"testing"
)

func TestCommandEnvNamesTheWorkingDirectoryAfterAnAppImageScrub(t *testing.T) {
	t.Setenv("APPIMAGE", "/home/me/mortar.appimage")
	t.Setenv("APPDIR", "/tmp/.mount_mortar")
	t.Setenv("PWD", "/tmp/.mount_mortar/usr/lib")
	got := commandEnv("/games/steamapps/common/Game", []string{"A=1"})
	if !slices.Contains(got, "PWD=/games/steamapps/common/Game") || !slices.Contains(got, "A=1") {
		t.Fatalf("got %v", got)
	}
	if slices.Contains(got, "PWD=/tmp/.mount_mortar/usr/lib") {
		t.Fatal("the mount's PWD reached the child")
	}
}
