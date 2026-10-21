package dotnet

import (
	"path/filepath"
	"slices"
	"testing"
)

// testdata/mod.dll's Fixture.Patches covers each way a patch names its target: class and method attributes merged, a
// method found by name, a getter and a constructor, argument types, and HarmonyX's type name as a string; Fixture.Hooks
// subscribes a HookGen On hook and an IL hook. Each attribute patch names its own method as a stack frame would.
func TestPatchesReadsHarmonyAttributes(t *testing.T) {
	got, err := Patches(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Patch{
		{"StardewValley.Farmer::.ctor", PatchPostfix, "Fixture.Patches.Built::Postfix"},
		{"StardewValley.Farmer::Update", PatchHook, ""},
		{"StardewValley.Farmer::Update", PatchSkip, "Fixture.Patches.FarmerPatches::SkipUpdate"},
		{"StardewValley.Farmer::get_CurrentToolIndex", PatchPostfix, "Fixture.Patches.FarmerPatches::Postfix"},
		{"StardewValley.Game1::Draw", PatchPrefix, "Fixture.Patches.DrawPatch::Prefix"},
		{"StardewValley.Game1::Draw", PatchTranspiler, "Fixture.Patches.DrawPatch::Transpiler"},
		{"StardewValley.Game1::Update", PatchTranspiler, ""},
		{"StardewValley.Menus.ClickableComponent::snap", PatchFinalizer, "Fixture.Patches.ByName::Done"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("patches = %v\nwant      %v", got, want)
	}
}
