package launch

import (
	"os"
	"strings"
	"testing"
)

func TestParseSMAPIProblemsRecognisesSMAPIWording(t *testing.T) {
	log := strings.Join([]string{
		"[19:43:50 ERROR SMAPI] Skipped mods because they need mods that aren't installed:",
		"[19:43:50 ERROR SMAPI]    - DeluxeGrabber 1.4.2 because it needs Pathoschild.ContentPatcher, which isn't installed.",
		"[19:43:50 ERROR SMAPI] Skipped 'Old Pack' because it needs SpaceCore, which isn't installed.",
		"[19:43:50 ERROR SMAPI] Automate is no longer compatible. Please update it for the latest version of Stardew Valley.",
		"[19:43:50 ERROR SMAPI]    - JSON Assets because it requires a newer version of SMAPI.",
		"[19:43:50 ERROR Content Patcher] Can't load content pack 'Festive Farm' because it doesn't have a valid manifest.",
		"[19:43:50 ERROR Content Patcher] Can't apply patch 'Maps/Town' from content pack 'Festive Farm': the FromFile file doesn't exist.",
		"[19:43:50 ERROR SMAPI]    - Lookup Anything because you have multiple copies of this mod installed (Mods/LookupAnything, Mods/LookupAnything (2)).",
		"[19:43:50 ERROR SMAPI] Failed loading the 'Broken Mod' mod. The error message from the code loader follows:",
		"[19:43:50 INFO SMAPI] Loaded 80 mods.",
	}, "\n")
	got := ParseSMAPIProblems(log)
	want := []struct {
		kind SMAPIProblemKind
		fix  SMAPIFixKind
		name string
		dep  string
	}{
		{SMAPIProblemMissingDependency, SMAPIFixInstallDependency, "DeluxeGrabber 1.4.2", "Pathoschild.ContentPatcher"},
		{SMAPIProblemMissingDependency, SMAPIFixInstallDependency, "Old Pack", "SpaceCore"},
		{SMAPIProblemTooOld, SMAPIFixUpdate, "Automate", ""},
		{SMAPIProblemTooOld, SMAPIFixUpdate, "JSON Assets", ""},
		{SMAPIProblemContentPack, SMAPIFixDisable, "Festive Farm", ""},
		{SMAPIProblemDuplicate, SMAPIFixRemoveDuplicate, "Lookup Anything", ""},
		{SMAPIProblemFailedToLoad, SMAPIFixDisable, "Broken Mod", ""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d findings %#v, want %d", len(got), got, len(want))
	}
	for i, w := range want {
		p := got[i]
		if p.Kind != w.kind || p.Fix != w.fix || p.ModName != w.name || p.Dependency != w.dep {
			t.Fatalf("finding %d: got kind=%s fix=%s name=%q dep=%q, want kind=%s fix=%s name=%q dep=%q (detail %q)",
				i, p.Kind, p.Fix, p.ModName, p.Dependency, w.kind, w.fix, w.name, w.dep, p.Detail)
		}
		if p.Detail == "" {
			t.Fatalf("finding %d has empty detail", i)
		}
	}
}

func TestParseSMAPIProblemsEmpty(t *testing.T) {
	got := ParseSMAPIProblems("[19:43:50 INFO SMAPI] Loaded 80 mods.\n")
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestResolveSMAPIProblemModsFillsUniqueID(t *testing.T) {
	problems := []SMAPIProblem{{
		Kind: SMAPIProblemTooOld, ModID: "Automate", ModName: "Automate", Fix: SMAPIFixUpdate, Detail: "x",
	}}
	got := ResolveSMAPIProblemMods(problems, []ModRef{{Name: "Automate", ID: "smapi:Pathoschild.Automate"}})
	if got[0].ModID != "Pathoschild.Automate" {
		t.Fatalf("got %#v", got)
	}
}

// TestParseSMAPIProblemsRealLogs reads lines from real SMAPI 4.x logs (home folder anonymised) and the failure
// phrases of SMAPI's ModResolver, so every cluster those logs hold maps to a problem with a fix.
func TestParseSMAPIProblemsRealLogs(t *testing.T) {
	logged, err := os.ReadFile("testdata/smapi-real.txt")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Join([]string{
		"[10:00:00 ERROR SMAPI]       - Old Map 1.0.0 because it needs SMAPI 4.9.0 or later. Please update SMAPI to the latest version to use this mod.",
		"[10:00:00 ERROR SMAPI]       - New Crops 2.0.0 because it needs Stardew Valley 1.7.0 or later. Please update your game to the latest version to use this mod.",
		"[10:00:00 ERROR SMAPI]       - Chests Anywhere Lite 1.0.0 because it's obsolete: its features are now part of the game.",
		"[10:00:00 ERROR SMAPI]       - Tractor Addon 1.0.0 because it needs newer versions of some mods: Tractor Mod (needs 4.16.0 or later).",
		"[10:00:00 ERROR SMAPI]       - Lookup Anything 1.40.0 because you have multiple copies of this mod installed. To fix this, delete these folders and reinstall the mod: Mods/LookupAnything, Mods/LookupAnything (2).",
		"[10:00:00 ERROR SMAPI]       - Bad Pack 1.0.0 because its DLL 'BadPack.dll' doesn't exist.",
		"[10:00:00 ERROR SMAPI] The 'Free Gold' mod has been flagged as a malicious mod.",
		"[10:00:00 ERROR Better Crafting] Patching exception in method virtual void StardewValley.Menus.CraftingPage::draw(SpriteBatch b)",
	}, "\n")
	got := ParseSMAPIProblems(string(logged) + "\n" + source)
	want := []struct {
		kind      SMAPIProblemKind
		fix       SMAPIFixKind
		name, dep string
	}{
		{SMAPIProblemMissingDependency, SMAPIFixInstallDependency, "Seed Alpha 1.0.0", "Content Patcher"},
		{SMAPIProblemMissingDependency, SMAPIFixInstallDependency, "Adventurer's Guild Expanded Music 1.0.0", "ZeroMeters.SAAT.Mod"},
		{SMAPIProblemTooOld, SMAPIFixUpdate, "SAAT.API 1.1.2", ""},
		{SMAPIProblemDependencyFailed, SMAPIFixUpdate, "SAAT.Mod 1.1.2", "SAAT.API"},
		{SMAPIProblemTooOld, SMAPIFixUpdate, "More Giant Crops 1.2.0", ""},
		{SMAPIProblemDuplicate, SMAPIFixRemoveDuplicate, "Help Wanted 0.9.3", ""},
		{SMAPIProblemFailedToLoad, SMAPIFixDisable, "Wedding Tweaks", ""},
		{SMAPIProblemHarmonyPatch, SMAPIFixUpdate, "DynamicShader", ""},
		{SMAPIProblemFailedToLoad, SMAPIFixDisable, "Love of Cooking", ""},
		{SMAPIProblemContentPack, SMAPIFixDisable, "Sharogg's Tilesheets", ""},
		{SMAPIProblemContentPack, SMAPIFixDisable, "Vegas' Item Compatibility Patch", ""},
		{SMAPIProblemContentPack, SMAPIFixDisable, "Ellianas Soft Hairstyles", ""},
		{SMAPIProblemContentPack, SMAPIFixDisable, "Cow Ears Accessory", ""},
		{SMAPIProblemSMAPITooOld, SMAPIFixUpdateLoader, "Old Map 1.0.0", ""},
		{SMAPIProblemGameTooOld, SMAPIFixDisable, "New Crops 2.0.0", ""},
		{SMAPIProblemObsolete, SMAPIFixDisable, "Chests Anywhere Lite 1.0.0", ""},
		{SMAPIProblemDependencyTooOld, SMAPIFixUpdate, "Tractor Addon 1.0.0", "Tractor Mod"},
		{SMAPIProblemDuplicate, SMAPIFixRemoveDuplicate, "Lookup Anything 1.40.0", ""},
		{SMAPIProblemFailedToLoad, SMAPIFixDisable, "Bad Pack 1.0.0", ""},
		{SMAPIProblemMalicious, SMAPIFixRemove, "Free Gold", ""},
		{SMAPIProblemHarmonyPatch, SMAPIFixUpdate, "Better Crafting", ""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d findings, want %d: %#v", len(got), len(want), got)
	}
	for i, w := range want {
		if p := got[i]; p.Kind != w.kind || p.Fix != w.fix || p.ModName != w.name || p.Dependency != w.dep {
			t.Errorf("finding %d = %s/%s %q dep %q, want %s/%s %q dep %q", i, p.Kind, p.Fix, p.ModName, p.Dependency, w.kind, w.fix, w.name, w.dep)
		}
	}
	// Only the mod's own chatter is left: Pet Against Crows' conflict notice.
	if n := Unclassified(string(logged)); n != 1 {
		t.Errorf("Unclassified = %d, want 1", n)
	}
}
