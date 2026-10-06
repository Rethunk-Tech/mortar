package steam

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestBackupBeforeEditOnlyCopiesOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "steam.vdf")
	if err := fsx.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := backupBeforeEdit(path, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteFile(path, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := backupBeforeEdit(path, []byte("newer"), 0o600); err != nil {
		t.Fatal(err)
	}
	body, err := fsx.ReadFile(filepath.Join(dir, "steam.vdf.mortar.bak"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "old" {
		t.Fatalf("backup = %q", body)
	}
}

func TestAddShortcutCreatesTheAccountsFirstShortcutsFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	users := `"users" { "76561198000000002" { "AccountName" "b" "MostRecent" "1" } }`
	if err := os.WriteFile(filepath.Join(root, "config", "loginusers.vdf"), []byte(users), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Steam{Root: root}.AddShortcut(Shortcut{Name: "Main (Stardew Valley)", Exe: "/opt/mortar", StartDir: "/opt"})
	if err != nil || res != Added {
		t.Fatalf("add = %v, %v", res, err)
	}
	dir := filepath.Join(root, "userdata", "39734274", "config")
	if _, err := os.Stat(filepath.Join(dir, "shortcuts.vdf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "shortcuts.vdf.mortar.bak")); !os.IsNotExist(err) {
		t.Fatalf("a file that did not exist has no backup: %v", err)
	}
}

func TestAtomicWriteFileLeavesOldFileOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "steam.vdf")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteFile(filepath.Join(dir, "missing", "steam.vdf"), []byte("new"), 0o600); err == nil {
		t.Fatal("expected write failure")
	}
	body, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "old" {
		t.Fatalf("old file changed: %q", body)
	}
}
