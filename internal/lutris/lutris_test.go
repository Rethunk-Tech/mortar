package lutris

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallFromFixture(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(filepath.Join("testdata", "gog-linux.yml"))
	if err != nil {
		t.Fatal(err)
	}
	replaced := strings.ReplaceAll(string(text), "/home/player/Games/Stardew Valley", dir)
	// strings.Replace
	got, err := installFromFile(writeTempYml(t, replaced))
	if err != nil || got != dir {
		t.Fatalf("install = %q, %v", got, err)
	}
	steam, err := installFromFile(filepath.Join("testdata", "steam-runner.yml"))
	if err != nil || steam != "" {
		t.Fatalf("steam runner = %q, %v", steam, err)
	}
}

func writeTempYml(t *testing.T, body string) string {
	path := filepath.Join(t.TempDir(), "game.yml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
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
	if err := os.WriteFile(filepath.Join(install, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	yml := "game_slug: stardew-valley\nscript:\n  runner: linux\ngame:\n  working_dir: " + install + "\n"
	if err := os.WriteFile(filepath.Join(games, "stardew.yml"), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	found := Locate(home)
	if len(found) != 1 || found[0].Dir != install {
		t.Fatalf("Locate = %+v", found)
	}
}
