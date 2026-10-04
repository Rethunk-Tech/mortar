package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConsoleRevealDirAllowsModsAndGameRoots(t *testing.T) {
	mods := t.TempDir()
	gameDir := t.TempDir()
	file := filepath.Join(mods, "FTM", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ConsoleRevealDir(file, []string{mods, gameDir})
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Dir(file) {
		t.Fatalf("got %s, want %s", got, filepath.Dir(file))
	}
	insideGame := filepath.Join(gameDir, "Content")
	if err := os.MkdirAll(insideGame, 0o750); err != nil {
		t.Fatal(err)
	}
	got, err = ConsoleRevealDir(insideGame, []string{mods, gameDir})
	if err != nil {
		t.Fatal(err)
	}
	if got != insideGame {
		t.Fatalf("dir = %s, want %s", got, insideGame)
	}
}

func TestConsoleRevealDirRejectsEscapes(t *testing.T) {
	mods := t.TempDir()
	gameDir := t.TempDir()
	outside := t.TempDir()
	file := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsoleRevealDir(file, []string{mods, gameDir}); err == nil {
		t.Fatal("outside path accepted")
	}
	escape := filepath.Join(mods, "..", filepath.Base(outside), "secret.txt")
	if _, err := ConsoleRevealDir(escape, []string{mods, gameDir}); err == nil {
		t.Fatal(".. escape accepted")
	}
}

func TestConsoleRevealDirRejectsSymlinkOut(t *testing.T) {
	mods := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "other")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(mods, "escape")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	if _, err := ConsoleRevealDir(link, []string{mods}); err == nil {
		t.Fatal("symlink out of mods accepted")
	}
}
