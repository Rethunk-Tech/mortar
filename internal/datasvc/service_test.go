package datasvc

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/testenv"
)

func TestRemoveStoreItemRefusesWhenAProfileUsesIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, profiles := testenv.Stores(t)
	p, err := profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	itemDir := filepath.Join(root, "store", "stardew", "nexus-1-1")
	if err := os.MkdirAll(itemDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(itemDir, "m.bin"), []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}
	profPath := filepath.Join(root, "profiles", "stardew", p.ID, "profile.json")
	setEntry(t, profPath, "nexus-1-1")
	svc := NewService(items, profiles, nil)
	if err := svc.RemoveStoreItem("stardew", "nexus-1-1"); !errors.Is(err, errInUse) {
		t.Fatalf("in use: %v", err)
	}
	if _, err := os.Stat(filepath.Join(itemDir, "m.bin")); err != nil {
		t.Fatal("in-use item was removed")
	}
	setEntry(t, profPath, "")
	if err := svc.RemoveStoreItem("stardew", "nexus-1-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(itemDir); !os.IsNotExist(err) {
		t.Fatalf("unused item remains: %v", err)
	}
}

func setEntry(t *testing.T, path, key string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if key == "" {
		raw["entries"] = []any{}
	} else {
		raw["entries"] = []any{map[string]any{"key": key, "mods": []any{}}}
	}
	out, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
}
