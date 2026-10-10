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

func TestInstallFromAConfigWhoseExeIsAFlatMarkerOrANestedMarkerInOtherCase(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	flat := Game{Slug: "stardew-valley", Marker: "Stardew Valley.dll"}
	got, err := newMatcher(flat).installFromYAML("game_slug: stardew-valley\ngame:\n  exe: " + filepath.Join(root, "Stardew Valley.dll") + "\n")
	if err != nil || got != root {
		t.Fatalf("flat marker: %q, %v", got, err)
	}
	nested := Game{Slug: "the-sims-4", Marker: "Game/Bin/TS4_x64.exe"}
	bin := filepath.Join(root, "Game", "Bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "TS4_x64.exe"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = newMatcher(nested).installFromYAML("game_slug: the-sims-4\ngame:\n  exe: " + filepath.Join(root, "game", "bin", "ts4_x64.EXE") + "\n")
	if err != nil || got != root {
		t.Fatalf("nested marker, other case: %q, %v", got, err)
	}
}
