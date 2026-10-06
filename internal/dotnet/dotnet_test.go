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

func TestScanReadsDependenciesAndIncompatibilitiesOfAPluginClass(t *testing.T) {
	got, err := Scan(filepath.Join("testdata", "mod.dll"))
	want := []Relation{
		{Plugin: "com.fixture.plugin", GUID: "com.fixture.hard", Kind: HardDependency},
		{Plugin: "com.fixture.plugin", GUID: "com.fixture.min", MinVersion: "2.1.0", Kind: HardDependency},
		{Plugin: "com.fixture.plugin", GUID: "com.fixture.soft", Kind: SoftDependency},
		{Plugin: "com.fixture.plugin", GUID: "com.fixture.clash", Kind: Incompatible},
	}
	if err != nil || len(got.Plugins) != 1 || !slices.Equal(got.Relations, want) {
		t.Fatalf("relations = %+v, %v", got.Relations, err)
	}
}

func TestRelationArgsTellsAMinimumVersionFromFlags(t *testing.T) {
	guid := []byte{1, 0, 1, 'g'}
	cases := map[string]struct {
		blob []byte
		want Relation
		ok   bool
	}{
		"flags hard":    {append(slices.Clone(guid), 1, 0, 0, 0, 0, 0), Relation{GUID: "g", Kind: HardDependency}, true},
		"flags soft":    {append(slices.Clone(guid), 2, 0, 0, 0, 0, 0), Relation{GUID: "g", Kind: SoftDependency}, true},
		"short minimum": {append(slices.Clone(guid), 3, '1', '.', '0', 0, 0), Relation{GUID: "g", MinVersion: "1.0", Kind: HardDependency}, true},
		"truncated":     {append(slices.Clone(guid), 9, '1'), Relation{}, false},
		"no prolog":     {[]byte{0, 0, 1, 'g'}, Relation{}, false},
	}
	for name, c := range cases {
		if got, ok := relationArgs(c.blob, false); ok != c.ok || got != c.want {
			t.Errorf("%s: %+v, %v", name, got, ok)
		}
	}
	if got, ok := relationArgs(append(slices.Clone(guid), 0, 0), true); !ok || got.Kind != Incompatible || got.GUID != "g" {
		t.Errorf("incompatibility: %+v, %v", got, ok)
	}
}
