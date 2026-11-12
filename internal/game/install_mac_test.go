package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

func TestPlatformOfMacBundle(t *testing.T) {
	info := components.GameInfo{Marker: "Game.exe", DarwinMarker: "Game.app/Contents/MacOS/Game"}
	dir := t.TempDir()
	exe := filepath.Join(dir, filepath.FromSlash(info.DarwinMarker))
	if err := os.MkdirAll(filepath.Dir(exe), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(exe, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := platformOf(info, dir, "darwin"); got != "darwin" {
		t.Errorf("darwin host: %q", got)
	}
	if got := platformOf(info, dir, "linux"); got != "windows" {
		t.Errorf("a Linux host does not read the bundle: %q", got)
	}
}

func TestPlatformOfReadsTheGamesMarkerBelowItsMarkerDir(t *testing.T) {
	info := components.GameInfo{Marker: "Game.exe", MarkerDir: "_windows/win64", LinuxMarker: "_linux/game.bin"}
	dir := t.TempDir()
	for _, f := range []string{"_windows/win64/Game.exe", "_linux/game.bin"} {
		p := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := safefs.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := platformOf(info, dir, "linux"); got != "windows" {
		t.Errorf("both builds installed: %q, want the Windows one the marker names", got)
	}
	if err := os.Remove(filepath.Join(dir, "_windows", "win64", "Game.exe")); err != nil {
		t.Fatal(err)
	}
	if got := platformOf(info, dir, "linux"); got != "linux" {
		t.Errorf("only the native build installed: %q", got)
	}
}
