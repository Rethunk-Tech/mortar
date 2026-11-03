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
	// exec.Cmd keeps the last value of a repeated variable, and only Linux scrubs the inherited PWD.
	if got[len(got)-1] != "PWD=/games/steamapps/common/Game" || !slices.Contains(got, "A=1") {
		t.Fatalf("got %v", got)
	}
}
