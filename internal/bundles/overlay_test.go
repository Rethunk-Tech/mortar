package bundles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	modstore "github.com/Rethunk-Tech/mortar/internal/store"
)

func addFiles(t *testing.T, items *modstore.Store, key string, files map[string]string) {
	t.Helper()
	src := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(src, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := items.AddDir(t.Context(), "stardew", key, src); err != nil {
		t.Fatal(err)
	}
}

func TestBundleCarriesOptionalFiles(t *testing.T) {
	svc, profiles, items := testService(t)
	main, opt := modstore.NexusKey(5, 1), modstore.NexusKey(5, 2)
	addFiles(t, items, main, map[string]string{"Mod/manifest.json": `{"Name":"M","UniqueID":"M.One","Version":"1.0.0"}`, "Mod/a.png": "main"})
	addFiles(t, items, opt, map[string]string{"Mod/a.png": "opt"})
	src := func(file int, name string) profile.Source {
		return profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: file, Name: name}
	}
	from, err := profiles.Create("stardew", "From")
	if err != nil {
		t.Fatal(err)
	}
	for _, add := range []struct {
		key  string
		file int
	}{{main, 1}, {opt, 2}} {
		if _, err := profiles.AddEntry("stardew", from.ID, add.key, src(add.file, add.key+".zip")); err != nil {
			t.Fatal(err)
		}
	}
	b, err := svc.Create("stardew", "Set", from.ID, []mod.ID{"smapi:M.One"})
	if err != nil || len(b.Mods) != 2 || b.Mods[1].OverlayOf != main || b.Mods[1].OverlayTo != "Mod" {
		t.Fatalf("bundle = %+v, %v", b.Mods, err)
	}
	to, err := profiles.Create("stardew", "To")
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.Apply("stardew", b.ID, to.ID)
	if err != nil || len(res.Profile.Entries) != 2 || res.Profile.Entries[1].OverlayOf != main {
		t.Fatalf("apply = %+v, %v", res.Profile.Entries, err)
	}
	dir, err := profiles.ModsDir("stardew", to.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.ReadFile(filepath.Join(dir, main, "Mod", "a.png")); string(got) != "opt" {
		t.Fatalf("laid file = %q", got)
	}
}
