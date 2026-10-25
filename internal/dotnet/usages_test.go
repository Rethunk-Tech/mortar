package dotnet

import (
	"path/filepath"
	"slices"
	"testing"
)

// testdata/mod.dll: a field read and written in two methods, a property setter, and a game member referenced from
// another assembly.
func TestUsagesFindsCallersAndFieldUses(t *testing.T) {
	path := filepath.Join("testdata", "mod.dll")
	for target, want := range map[string][]string{
		"Fixture.Mod::own":                           {"Fixture.Mod::SwitchTool", "Fixture.Mod::Read"},
		"Fixture.Mod::set_Own":                       {"Fixture.Mod::SwitchTool"},
		"StardewValley.Farmer::set_CurrentToolIndex": {"Fixture.Mod::SwitchTool"},
		"StardewValley.Game1::Update":                {},
	} {
		got, err := Usages(path, target)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: %v %v", target, got, err)
		}
	}
}
