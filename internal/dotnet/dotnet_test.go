package dotnet

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// testdata/mod.dll is compiled from testdata/src (rebuild with testdata/src/build.sh). Its BuffEffects writes and the
// myID and leftNeighborID stores go to objects the mod builds, so they are left out.
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
		"StardewValley.Menus.ClickableComponent::rightNeighborID",
		"StardewValley.Menus.ClickableComponent::upNeighborID",
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

func TestPluginArgsReadsTheAttributeBlob(t *testing.T) {
	blob := []byte{1, 0, 8, 'a', '.', 'b', '.', 'c', 'd', 'e', 'f', 4, 'N', 'a', 'm', 'e', 5, '1', '.', '0', '.', '0', 0, 0}
	got, ok := pluginArgs(blob)
	if !ok || got != (Plugin{GUID: "a.b.cdef", Name: "Name", Version: "1.0.0"}) {
		t.Fatalf("pluginArgs = %+v, %v", got, ok)
	}
	if _, ok := pluginArgs(blob[:12]); ok {
		t.Fatal("truncated blob accepted")
	}
}

func TestPluginsReadsTheAttributeAndItsClassNamespace(t *testing.T) {
	got, err := Plugins(filepath.Join("testdata", "mod.dll"))
	want := []Plugin{{GUID: "com.fixture.plugin", Name: "Fixture Plugin", Version: "1.2.3", Namespace: "Fixture.Plugins"}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("got %+v, %v", got, err)
	}
}
