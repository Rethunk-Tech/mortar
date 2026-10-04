package dotnet

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// testdata/mod.dll is compiled from testdata/src (rebuild with testdata/src/build.sh).
func TestWritesListsGameMembersTheAssemblyAssigns(t *testing.T) {
	got, err := Writes(filepath.Join("testdata", "mod.dll"), "StardewValley", "Netcode")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"StardewValley.Farmer+Pocket::Count",
		"StardewValley.Farmer::CurrentToolIndex",
		"StardewValley.Farmer::NetMoney",
		"StardewValley.Farmer::health",
		"StardewValley.Farmer::stamina",
		"StardewValley.Game1::flashAlpha",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("writes = %q\nwant     %q", got, want)
	}
}

func TestWritesRejectsFilesThatAreNotAssemblies(t *testing.T) {
	full, err := fsx.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for i, data := range [][]byte{[]byte("not a PE file"), full[:len(full)/2]} {
		path := filepath.Join(dir, "mod.dll")
		if err := fsx.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if got, err := Writes(path, "StardewValley", "Netcode"); err == nil {
			t.Fatalf("case %d: writes = %q, want an error", i, got)
		}
	}
}
