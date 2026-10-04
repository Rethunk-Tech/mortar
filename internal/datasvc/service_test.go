package datasvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
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
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	// A bundle-only and a template-only local item: no profile names either.
	for _, key := range []string{"local-bundle", "local-template"} {
		dir := filepath.Join(root, "store", "stardew", key)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "m.bin"), []byte("mod"), 0o600); err != nil {
			t.Fatal(err)
		}
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
		if _, err := os.Stat(filepath.Join(root, "store", "stardew", key, "m.bin")); err != nil {
			t.Fatalf("collect removed %s: %v", key, err)
		}
	}
}
