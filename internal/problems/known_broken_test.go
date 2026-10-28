package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestCuratedBrokenNamesPackagesAndModsThroughTheirListedVersion(t *testing.T) {
	t.Parallel()
	list := components.BrokenList{Entries: []components.BrokenEntry{
		{IDs: components.BrokenIDs{Thunderstore: "Ns-OldLib"}, Through: "2.0.0", Reason: "Crashes on load.", Replacement: "Ns-NewLib"},
		{IDs: components.BrokenIDs{SMAPI: "Me.Mod"}, Through: "1.2.0", Reason: "Breaks saves."},
	}}
	pkgs := []profile.PackageRef{
		{Key: "a", ID: mod.ID("thunderstore:Ns-OldLib"), Name: "Ns-OldLib", Version: "2.0.0"},
		{Key: "b", ID: mod.ID("thunderstore:Ns-OldLib"), Name: "Ns-OldLib", Version: "2.0.1"},
	}
	mods := []framework.Mod{{Key: "d", Name: "Me.Mod", UniqueID: "Me.Mod", Version: "1.2.0"}}
	got := curatedBroken(list, "", "", pkgs, mods)
	keys := map[string]Broken{}
	for _, b := range got {
		keys[b.Key] = b
	}
	if len(got) != 2 || keys["a"].Replacement == nil || keys["a"].Replacement.PageName != "NewLib" || keys["a"].Replacement.Package != "Ns-NewLib" || keys["a"].Replacement.Site != "thunderstore" {
		t.Fatalf("rows = %+v", got)
	}
	if _, fixed := keys["b"]; fixed {
		t.Error("a version past the list's was flagged")
	}
	for _, b := range got {
		if b.Status != "broken" || b.Source != "Mortar" || b.Summary == "" {
			t.Errorf("row %+v", b)
		}
	}
}

func TestWithKnownBrokenMarksStillListedAndFixingUpdates(t *testing.T) {
	t.Parallel()
	list := components.BrokenList{Entries: []components.BrokenEntry{
		{IDs: components.BrokenIDs{Thunderstore: "Ns-Lib"}, Through: "2.0.0", Reason: "Crashes on load."},
	}}
	r := withKnownBroken(list, "", UpdatesResult{Updates: []Update{
		{Key: "still", ID: mod.ID("thunderstore:Ns-Lib"), Installed: "1.0.0", Version: "2.0.0"},
		{Key: "fixes", ID: mod.ID("thunderstore:Ns-Lib"), Installed: "2.0.0", Version: "2.1.0"},
		{Key: "other", ID: mod.ID("thunderstore:Ns-Other"), Installed: "1.0.0", Version: "2.0.0"},
	}})
	still, fixes, other := r.Updates[0], r.Updates[1], r.Updates[2]
	if still.KnownBroken == nil || still.KnownBroken.Reason != "Crashes on load." || still.FixesKnownBroken {
		t.Errorf("still listed = %+v", still)
	}
	if fixes.KnownBroken != nil || !fixes.FixesKnownBroken {
		t.Errorf("fixing = %+v", fixes)
	}
	if other.KnownBroken != nil || other.FixesKnownBroken {
		t.Errorf("unlisted = %+v", other)
	}
}
