package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestPluginGUIDsOfFolderPackageAndUnknownMods(t *testing.T) {
	dll, err := os.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	e := newEnv(t)
	p, err := e.Create("lethal-company", "Lobby")
	if err != nil {
		t.Fatal(err)
	}
	addPackage(t, e, p, "Ns-Plug", map[string]string{"mod.dll": string(dll)})
	p, err = e.read("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	folder := mod.NewID(mod.FormatThunderstore, "Ns-Folder")
	p.Entries = append(p.Entries, Entry{Key: "ns-folder", Mods: []Component{{ID: folder, ModID: "x"}}})

	for _, c := range []struct {
		name string
		id   mod.ID
		want []string
	}{
		{"folder game mod", folder, []string{"x"}},
		{"unknown id", mod.NewID(mod.FormatThunderstore, "Nobody-Nothing"), nil},
		{"BepInEx package", mod.NewID(mod.FormatThunderstore, "Ns-Plug"), []string{"com.fixture.plugin"}},
	} {
		if got := e.pluginGUIDsOf("lethal-company", p, c.id); !slices.Equal(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestHoldDroppedMovesFilesUnderChangedAndMarksThemHeld(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, filepath.Join(oldFilesDir, "k", droppedFilesDir, "a", "b.txt"), "x")
	if err := holdDropped(dir, "k"); err != nil {
		t.Fatal(err)
	}
	if b, err := safefs.ReadFile(filepath.Join(dir, changedDir, "a", "b.txt")); err != nil || string(b) != "x" {
		t.Fatalf("changed copy %q, %v", b, err)
	}
	if _, ok := readHeld(dir)["a/b.txt"]; !ok {
		t.Fatalf("not marked held: %v", readHeld(dir))
	}
	if err := holdDropped(dir, "missing"); err != nil {
		t.Fatalf("missing root: %v", err)
	}
}
