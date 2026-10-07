package launchsvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func installed(key, id, name, dll string, enabled bool) profile.Installed {
	m := profile.Installed{Key: key, Enabled: enabled}
	m.UniqueID, m.Name, m.EntryDll = id, name, dll
	return m
}

func TestBlamedModMatchesNamespaceToAssemblyOrID(t *testing.T) {
	mods := []profile.Installed{
		installed("cp", "Pathoschild.ContentPatcher", "Content Patcher", "ContentPatcher.dll", true),
		installed("ftm", "Esca.FarmTypeManager", "Farm Type Manager", "FarmTypeManager.dll", true),
		installed("off", "Some.Disabled", "Disabled", "Disabled.dll", false),
	}
	blames := []loader.Blame{
		{Exception: "early", Namespace: "ContentPatcher.Framework"},
		{Exception: "late", Frame: "at FarmTypeManager.ModEntry.SpawnTimer()", Namespace: "FarmTypeManager"},
	}
	m, b, ok := blamedMod(blames, mods)
	if !ok || m.Key != "ftm" || b.Exception != "late" {
		t.Fatalf("culprit = %+v, %+v, %v", m, b, ok)
	}
	if _, _, ok := blamedMod([]loader.Blame{{Namespace: "Disabled"}, {Namespace: "Stardew.Mods"}}, mods); ok {
		t.Fatal("a disabled mod and a generic namespace blame nothing")
	}
}

func TestBlamedModFollowsAHarmonyIDAndSkipsAmbiguity(t *testing.T) {
	mods := []profile.Installed{
		installed("a", "Pathoschild.ChestsAnywhere", "Chests Anywhere", "ChestsAnywhere.dll", true),
		installed("b", "Esca.FarmTypeManager", "Farm Type Manager", "FarmTypeManager.dll", true),
		installed("c", "Other.FarmTypeManager", "Farm Type Manager 2", "FarmTypeManager.dll", true),
	}
	m, _, ok := blamedMod([]loader.Blame{{PatchOwner: "pathoschild.chestsanywhere"}}, mods)
	if !ok || m.Key != "a" {
		t.Fatalf("patch owner = %+v, %v", m, ok)
	}
	if _, _, ok := blamedMod([]loader.Blame{{Namespace: "FarmTypeManager"}}, mods); ok {
		t.Fatal("two mods sharing an assembly name blame neither")
	}
}
