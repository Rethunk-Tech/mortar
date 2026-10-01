package lutris

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

//go:embed testdata/gog-linux.yml
var fixtureGogLinux []byte

//go:embed testdata/steam-runner.yml
var fixtureSteamRunner []byte

func TestInstallFromFixture(t *testing.T) {
	dir := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(dir, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	replaced := strings.ReplaceAll(string(fixtureGogLinux), "/home/player/Games/Stardew Valley", dir)
	got, err := installFromYAML(replaced)
	if err != nil || got != dir {
		t.Fatalf("install = %q, %v", got, err)
	}
	steam, err := installFromYAML(string(fixtureSteamRunner))
	if err != nil || steam != "" {
		t.Fatalf("steam runner = %q, %v", steam, err)
	}
}

func TestLocateLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lutris discovery is Linux-only")
	}
	home := t.TempDir()
	games := filepath.Join(home, ".local", "share", "lutris", "games")
	if err := os.MkdirAll(games, 0o700); err != nil {
		t.Fatal(err)
	}
	install := filepath.Join(home, "sdv")
	if err := os.MkdirAll(install, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(install, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	yml := "game_slug: stardew-valley\nscript:\n  runner: linux\ngame:\n  working_dir: " + install + "\n"
	if err := fsx.WriteFile(filepath.Join(games, "stardew.yml"), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	found := Locate(home)
	if len(found) != 1 || found[0].Dir != install {
		t.Fatalf("Locate = %+v", found)
	}
}
