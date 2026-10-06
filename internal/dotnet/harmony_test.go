package dotnet

import (
	"path/filepath"
	"slices"
	"testing"
)

// testdata/mod.dll's Fixture.Patches covers each way a patch names its target: class and method attributes merged, a
// method found by name, a getter and a constructor, argument types, and HarmonyX's type name as a string.
func TestPatchesReadsHarmonyAttributes(t *testing.T) {
	got, err := Patches(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Patch{
		{"StardewValley.Farmer::.ctor", PatchPostfix},
		{"StardewValley.Farmer::Update", PatchSkip},
		{"StardewValley.Farmer::get_CurrentToolIndex", PatchPostfix},
		{"StardewValley.Game1::Draw", PatchPrefix},
		{"StardewValley.Game1::Draw", PatchTranspiler},
		{"StardewValley.Menus.ClickableComponent::snap", PatchFinalizer},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("patches = %v\nwant      %v", got, want)
	}
}
