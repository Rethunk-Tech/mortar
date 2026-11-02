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
