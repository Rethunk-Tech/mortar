package migrate

import (
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/syndtr/goleveldb/leveldb"
)

// newVortexHome builds a home whose Vortex config dir holds a real LevelDB at
// state.v2 with Vortex-shaped keys, plus a custom staging folder with two mods.
func newVortexHome(t *testing.T, pattern string) (home, state, mods string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "home")
	state = filepath.Join(configDir(home), "Vortex", "state.v2")
	mods = filepath.Join(root, "staging")
	if pattern != "custom" {
		mods = filepath.Join(configDir(home), "Vortex", "stardewvalley", "mods")
	}
	for name, id := range map[string]string{"Enabled-1": "Example.Enabled", "Disabled-1": "Example.Disabled"} {
		dir := filepath.Join(mods, name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		body := `{"Name":"` + name + `","UniqueID":"` + id + `","Version":"1.2.3"}`
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(state, 0o750); err != nil {
		t.Fatal(err)
	}
	db, err := leveldb.OpenFile(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	kvs := map[string]string{
		"persistent###profiles###p1###id":                                  `"p1"`,
		"persistent###profiles###p1###gameId":                              `"stardewvalley"`,
		"persistent###profiles###p1###name":                                `"Farm"`,
		"persistent###profiles###p1###modState###enabled-1###enabled":      `true`,
		"persistent###profiles###p1###modState###disabled-1###enabled":     `false`,
		"persistent###mods###stardewvalley###enabled-1###id":               `"enabled-1"`,
		"persistent###mods###stardewvalley###enabled-1###installationPath": `"Enabled-1"`,
		"persistent###mods###stardewvalley###disabled-1":                   `{"id":"disabled-1","installationPath":"Disabled-1"}`,
	}
	switch pattern {
	case "custom":
		kvs["settings###mods###installPath###stardewvalley"] = `"` + filepath.ToSlash(mods) + `"`
	case "":
	default:
		kvs["settings###mods###installPath###stardewvalley"] = pattern
	}
	for k, v := range kvs {
		if err := db.Put([]byte(k), []byte(v), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return home, state, mods
}

func hashTree(t *testing.T, dir string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fsx.ReadFile(path)
		out[path] = sha256.Sum256(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDetectsVortexLevelDBAndPreviewsMods(t *testing.T) {
	home, state, mods := newVortexHome(t, "custom")
	before := hashTree(t, state)

	// A second handle holding the source LOCK must not block the read.
	holder, err := fsx.OpenFile(filepath.Join(state, "LOCK"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Close() }()

	sources, err := Detect(home, "", "stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Kind != KindVortex || len(sources[0].Profiles) != 1 {
		t.Fatalf("sources = %#v", sources)
	}
	preview, err := Preview(home, "", "stardew", KindVortex, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if preview.ModsPath != filepath.Clean(mods) || len(preview.Mods) != 2 {
		t.Fatalf("preview = %#v", preview)
	}
	byID := map[string]ModPreview{}
	for _, m := range preview.Mods {
		byID[string(m.ID)] = m
	}
	on, off := byID["smapi:Example.Enabled"], byID["smapi:Example.Disabled"]
	if !on.Enabled || off.Enabled || off.ID == "" {
		t.Fatalf("mods = %#v", preview.Mods)
	}
	if on.SourcePath != filepath.Join(mods, "Enabled-1") || off.SourcePath != filepath.Join(mods, "Disabled-1") {
		t.Fatalf("source paths = %q, %q", on.SourcePath, off.SourcePath)
	}

	after := hashTree(t, state)
	if len(after) != len(before) {
		t.Fatalf("source files changed: %d -> %d", len(before), len(after))
	}
	for path, sum := range before {
		if after[path] != sum {
			t.Fatalf("%s modified", path)
		}
	}
}

func TestVortexStagingFolderDefaultsAndPlaceholders(t *testing.T) {
	cases := map[string]string{
		"unset":       "",
		"placeholder": `"{USERDATA}\\{game}\\mods"`,
	}
	for name, pattern := range cases {
		t.Run(name, func(t *testing.T) {
			home, _, mods := newVortexHome(t, pattern)
			preview, err := Preview(home, "", "stardew", KindVortex, "p1")
			if err != nil {
				t.Fatal(err)
			}
			if preview.ModsPath != mods || len(preview.Mods) != 2 {
				t.Fatalf("modsPath = %q want %q, mods = %#v", preview.ModsPath, mods, preview.Mods)
			}
		})
	}
}
