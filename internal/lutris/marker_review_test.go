package lutris

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// A game whose marker sits below the install root ("Game/Bin/TS4_x64.exe") is launched by a Lutris config whose exe is
// that marker file; the install is the root two folders above it, not the folder holding the executable.
func TestInstallFromAConfigWhoseExeIsANestedMarker(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "Game", "Bin", "TS4_x64.exe")
	if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(exe, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	g := Game{Slug: "the-sims-4", Keyword: "sims", Marker: "Game/Bin/TS4_x64.exe"}
	got, err := newMatcher(g).installFromYAML("game_slug: the-sims-4\ngame:\n  exe: " + exe + "\n")
	if err != nil || got != root {
		t.Fatalf("install = %q, %v; want %q", got, err, root)
	}
}
