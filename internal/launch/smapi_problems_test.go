package launch

import (
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
	got := ResolveSMAPIProblemMods(problems, []ModRef{{Name: "Automate", UniqueID: "Pathoschild.Automate"}})
	if got[0].ModID != "Pathoschild.Automate" {
		t.Fatalf("got %#v", got)
	}
}
