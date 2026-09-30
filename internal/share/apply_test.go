package share

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/profile"
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
	entries := []profile.Entry{{
		Key:  "nexus-1-2",
		Mods: []profile.EntryMod{{UniqueID: "A.Mod", Folder: "Inner"}, {UniqueID: "A.Gone", Folder: "Missing"}, {UniqueID: "A.Esc", Folder: "../x"}},
	}}
	configs := []Config{
		{UniqueID: "a.mod", Path: "config.json", Data: []byte(`{"a":1}`)},
		{UniqueID: "A.Mod", Path: "data/deep.json", Data: []byte(`{"b":2}`)},
		{UniqueID: "A.Mod", Path: "../escape.json", Data: []byte("x")},
		{UniqueID: "A.Mod", Path: "/abs.json", Data: []byte("x")},
		{UniqueID: "A.Mod", Path: "run.sh", Data: []byte("x")},
		{UniqueID: "A.Mod", Path: "manifest.json", Data: []byte("x")},
		{UniqueID: "A.Gone", Path: "config.json", Data: []byte("x")},
		{UniqueID: "A.Esc", Path: "config.json", Data: []byte("x")},
		{UniqueID: "Other.Mod", Path: "config.json", Data: []byte("x")},
	}
	written, err := Apply(mods, entries, configs)
	if err != nil || len(written) != 1 || written[0] != "A.Mod" {
		t.Fatalf("written = %v, %v", written, err)
	}
	for path, want := range map[string]string{"Inner/config.json": `{"a":1}`, "Inner/data/deep.json": `{"b":2}`} {
		got, err := fsx.ReadFile(filepath.Join(mods, "nexus-1-2", path))
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
	if walkErr != nil || len(files) != 2 {
		t.Errorf("files on disk = %v, %v", files, walkErr)
	}
}
