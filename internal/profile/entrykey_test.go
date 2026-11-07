package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const escapingKey = "../../../../VICTIM"

func backupWithKey(t *testing.T, e env, key string) (Profile, map[string][]byte) {
	t.Helper()
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	p := mustCreate(t, e, "Farm")
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	src, files, err := e.Backup("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	src.Entries[0].Key = key
	return src, files
}

func TestRestoreRefusesAnEntryKeyThatLeavesTheModsFolder(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	src, files := backupWithKey(t, e, escapingKey)
	victim := filepath.Join(filepath.Dir(e.root), "VICTIM")
	if err := os.MkdirAll(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := newEnv(t).RestoreBackup(t.Context(), "stardew", src, files, nil); err == nil {
		t.Fatal("a backup entry keyed outside the mods folder was restored")
	}
}

func TestLoadDropsAnEntryWhoseKeyLeavesTheModsFolder(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(e.root, "stardew", p.ID, fileName)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	evil := []byte(string(b))
	evil = []byte(strings.Replace(string(evil), `"key": "local-a"`, `"key": "`+escapingKey+`"`, 1))
	if string(evil) == string(b) {
		t.Fatal("the key was not rewritten")
	}
	if err := os.WriteFile(path, evil, 0o600); err != nil {
		t.Fatal(err)
	}
	forgetProfile(path)
	got, err := e.Mods("stardew", p.ID)
	_ = got
	if err != nil {
		t.Fatal(err)
	}
	read, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, en := range read.Entries {
		if en.Key == escapingKey {
			t.Fatal("the escaping entry survived loading")
		}
	}
}

func TestRemoveEntryFoldersRefusesAKeyOutsideTheModsFolder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	victim := filepath.Join(dir, "VICTIM")
	if err := os.MkdirAll(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := removeEntryFolders(filepath.Join(dir, "a", "mods"), "../../VICTIM"); err == nil {
		t.Fatal("removeEntryFolders accepted a key leaving the mods folder")
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("victim removed: %v", err)
	}
}
