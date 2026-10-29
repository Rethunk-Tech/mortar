package lutris

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
)

//go:embed testdata/gog-linux.yml
var fixtureGogLinux []byte

//go:embed testdata/steam-runner.yml
var fixtureSteamRunner []byte

// testGame is a game the way a Game implementation describes it to the Lutris locator.
var testGame = Game{Slug: "stardew-valley", Keyword: "stardew", Marker: "Stardew Valley.dll"}

func TestInstallFromFixture(t *testing.T) {
	dir := t.TempDir()
	if err := safefs.WriteFile(filepath.Join(dir, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	replaced := strings.ReplaceAll(string(fixtureGogLinux), "/home/player/Games/Stardew Valley", dir)
	got := newMatcher(testGame).installFromYAML(replaced)
	if got != dir {
		t.Fatalf("install = %q", got)
	}
	steam := newMatcher(testGame).installFromYAML(string(fixtureSteamRunner))
	if steam != "" {
		t.Fatalf("steam runner = %q", steam)
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
	if err := safefs.WriteFile(filepath.Join(install, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	yml := "game_slug: stardew-valley\nscript:\n  runner: linux\ngame:\n  working_dir: " + install + "\n"
	if err := safefs.WriteFile(filepath.Join(games, "stardew.yml"), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	found := Locate(home, testGame)
	if len(found) != 1 || found[0].Dir != install {
		t.Fatalf("Locate = %+v", found)
	}
}

func TestConfigDirsFollowCustomXDGHomes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // userdirs honours XDG and APPDATA only for os.UserHomeDir, which reads USERPROFILE on Windows
	cfg, data := filepath.Join(t.TempDir(), "cfg"), filepath.Join(t.TempDir(), "data")
	want := []string{filepath.Join(data, "lutris", "games"), filepath.Join(cfg, "lutris", "games")}
	check := func(t *testing.T) {
		t.Helper()
		got := ConfigDirs(home)
		for _, w := range want {
			if !slices.Contains(got, w) {
				t.Fatalf("%q missing from %v", w, got)
			}
		}
	}
	t.Run("native", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", cfg)
		t.Setenv("XDG_DATA_HOME", data)
		check(t)
	})
	t.Run("flatpak", func(t *testing.T) {
		t.Setenv("FLATPAK_ID", appversion.AppID)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "own", "config"))
		t.Setenv("XDG_DATA_HOME", filepath.Join(home, "own", "data"))
		t.Setenv("HOST_XDG_CONFIG_HOME", cfg)
		t.Setenv("HOST_XDG_DATA_HOME", data)
		check(t)
	})
}
