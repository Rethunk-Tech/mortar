package game

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestProtonDirsFindsProtonBehindAPrefix(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Proton and the game sandbox run only on Linux")
	}
	root := t.TempDir()
	target := filepath.Join(root, "elsewhere", "GE-Proton")
	if err := os.MkdirAll(target, 0o750); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "compatibilitytools.d", "GE-Proton")
	if err := os.MkdirAll(filepath.Dir(linked), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, linked); err != nil {
		t.Fatal(err)
	}
	proton := filepath.Join(linked, "proton")
	for name, cmd := range map[string]*launch.Command{
		"proton itself":   {Name: proton, Args: []string{"run", "game.exe"}},
		"behind a prefix": {Name: "/usr/bin/gamemoderun", Args: []string{proton, "run", "game.exe"}},
	} {
		got := protonDirs(cmd)
		if !slices.Contains(got, linked) {
			t.Fatalf("%s: %v, want %s", name, got, linked)
		}
	}
	if got := protonDirs(&launch.Command{Name: "/usr/bin/game", Args: []string{"proton"}}); len(got) != 0 {
		t.Fatalf("a bare word named proton: %v", got)
	}
}
