package share

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestApplyWritesOnlyValidConfigsInsideModFolders(t *testing.T) {
	mods := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(mods, "nexus-1-2", "Inner"), 0o750))
	must(os.MkdirAll(filepath.Join(mods, "nexus-1-2", "Pack", ".Off"), 0o750))
	must(os.MkdirAll(filepath.Join(mods, ".nexus-3-4"), 0o750))
	entries := []profile.Entry{{
		Key:  "nexus-1-2",
		Mods: []profile.Component{{ID: "smapi:A.Mod", Folder: "Inner"}, {ID: "smapi:A.Gone", Folder: "Missing"}, {ID: "smapi:A.Esc", Folder: "../x"}, {ID: "smapi:A.Off", Folder: "Pack/Off"}},
	}, {Key: "nexus-3-4", Mods: []profile.Component{{ID: "smapi:A.Root", Folder: "."}}}}
	configs := []Config{
		{ID: "smapi:a.mod", Path: "config.json", Data: []byte(`{"a":1}`)},
		{ID: "smapi:A.Mod", Path: "data/deep.json", Data: []byte(`{"b":2}`)},
		{ID: "smapi:A.Mod", Path: "../escape.json", Data: []byte("x")},
		{ID: "smapi:A.Mod", Path: "/abs.json", Data: []byte("x")},
		{ID: "smapi:A.Mod", Path: "run.sh", Data: []byte("x")},
		{ID: "smapi:A.Mod", Path: "manifest.json", Data: []byte("x")},
		{ID: "smapi:A.Gone", Path: "config.json", Data: []byte("x")},
		{ID: "smapi:A.Esc", Path: "config.json", Data: []byte("x")},
		{ID: "smapi:Other.Mod", Path: "config.json", Data: []byte("x")},
		{ID: "smapi:A.Off", Path: "config.json", Data: []byte("off")},
		{ID: "smapi:A.Root", Path: "config.json", Data: []byte("root")},
	}
	written, err := Apply(mods, entries, configs)
	if err != nil || strings.Join(mod.Locals(written), ",") != "A.Mod,A.Off,A.Root" {
		t.Fatalf("written = %v, %v", written, err)
	}
	for path, want := range map[string]string{
		"nexus-1-2/Inner/config.json": `{"a":1}`, "nexus-1-2/Inner/data/deep.json": `{"b":2}`,
		"nexus-1-2/Pack/.Off/config.json": "off", ".nexus-3-4/config.json": "root",
	} {
		got, err := fsx.ReadFile(filepath.Join(mods, path))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v", path, got, err)
		}
	}
	var files []string
	walkErr := filepath.WalkDir(mods, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, p)
		}
		return err
	})
	if walkErr != nil || len(files) != 4 {
		t.Errorf("files on disk = %v, %v", files, walkErr)
	}
}
