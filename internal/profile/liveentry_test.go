package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// A restored config goes to the entry's plain folder when it has no folder yet: a dotted one would switch its mods off.
func TestApplyRestoredWritesPlainFolderWhenNoEntryFolder(t *testing.T) {
	dir, modsDir := t.TempDir(), t.TempDir()
	src := filepath.Join(dir, restoreDir, "someMod")
	if err := os.MkdirAll(src, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, configFile), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := applyRestored(dir, modsDir, "someMod"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(modsDir, "someMod", configFile)); err != nil {
		t.Fatalf("restored config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(modsDir, ".someMod")); !os.IsNotExist(err) {
		t.Fatalf("restored config made a dotted folder: %v", err)
	}
}

func TestEntryDirExists(t *testing.T) {
	modsDir := t.TempDir()
	if entryDirExists(modsDir, "on") || entryDirExists(modsDir, "off") {
		t.Fatal("no folder should not exist")
	}
	if err := os.Mkdir(filepath.Join(modsDir, "on"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(modsDir, ".off"), 0o700); err != nil {
		t.Fatal(err)
	}
	if !entryDirExists(modsDir, "on") || !entryDirExists(modsDir, "off") {
		t.Fatal("a folder switched on or off should exist")
	}
}
