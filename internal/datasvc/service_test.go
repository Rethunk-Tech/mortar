package datasvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

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

func TestKeepSetSourcesProtectItemsFromCleanupAndCollect(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, profiles := testenv.Stores(t)
	// A bundle-only and a template-only local item: no profile names either.
	for _, key := range []string{"local-bundle", "local-template"} {
		addItem(t, items, key, map[string]string{"m.bin": "mod"})
	}
	sources := []KeySource{
		func() (map[string][]string, error) { return map[string][]string{"stardew": {"local-bundle"}}, nil },
		func() (map[string][]string, error) { return map[string][]string{"stardew": {"local-template"}}, nil },
	}
	svc := NewService(items, profiles, sources)
	preview, err := svc.CleanupPreview()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range preview.Items {
		if it.Kind == "store" {
			t.Fatalf("referenced item listed for cleanup: %+v", it)
		}
	}
	keys, err := KeepSet(profiles, false, sources)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, at := range []time.Time{now, now.Add(90 * 24 * time.Hour)} {
		if err := items.Collect(keys, at); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{"local-bundle", "local-template"} {
		if _, err := items.Dir("stardew", key); err != nil {
			t.Fatalf("collect removed %s: %v", key, err)
		}
	}
}

// addItem stores a folder of the given files under key.
func addItem(t *testing.T, items *store.Store, key string, files map[string]string) {
	t.Helper()
	src := t.TempDir()
	for rel, body := range files {
		testfs.WriteFile(t, src, rel, body)
	}
	if err := items.AddDir("stardew", key, src); err != nil {
		t.Fatal(err)
	}
}
