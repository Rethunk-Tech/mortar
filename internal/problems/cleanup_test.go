package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestCleanupHintsFindsUnusedFrameworkNeededOnlyByDisabledMod(t *testing.T) {
	spaceCore := smapimanifest.Manifest{
		Name:     "SpaceCore",
		UniqueID: "spacechase0.SpaceCore",
	}
	oldPack := smapimanifest.Manifest{
		UniqueID: "Example.OldPack",
		Dependencies: []mod.Dependency{{
			UniqueID: "spacechase0.SpaceCore",
			Required: true,
		}},
	}
	contentFramework := smapimanifest.Manifest{
		Name:     "Content Framework",
		UniqueID: "Example.ContentFramework",
	}
	oldContentPack := smapimanifest.Manifest{
		UniqueID:       "Example.OldContentPack",
		ContentPackFor: "Example.ContentFramework",
	}
	mods := []framework.Mod{
		{
			Key:      "spacecore",
			Enabled:  true,
			Manifest: spaceCore,
		},
		{
			Key:      "old-pack",
			Enabled:  false,
			Manifest: oldPack,
		},
		{
			Key:      "content-framework",
			Enabled:  true,
			Manifest: contentFramework,
		},
		{
			Key:      "old-content-pack",
			Enabled:  false,
			Manifest: oldContentPack,
		},
	}

	got := cleanupHints(mods, nil, nil)
	if len(got) != 2 || got[0].Key != "content-framework" || got[1].Key != "spacecore" {
		t.Fatalf("cleanupHints() = %#v, want both unused frameworks", got)
	}
	// A framework the player builds in a Develop project stays, even when only switched-off mods need it.
	mods[0].SourceFolder = "/projects/SpaceCore/artifacts/bin/SpaceCore/debug"
	if got := cleanupHints(mods, nil, nil); len(got) != 1 || got[0].Key != "content-framework" {
		t.Fatalf("cleanupHints() = %#v, want the developed framework kept", got)
	}
}

func TestCleanupHintsIgnoresOptionalDependents(t *testing.T) {
	worldMaps := smapimanifest.Manifest{Name: "World Maps", UniqueID: "Example.WorldMaps"}
	earthyMaps := smapimanifest.Manifest{
		UniqueID:     "Example.EarthyMaps",
		Dependencies: []mod.Dependency{{UniqueID: "Example.WorldMaps", Required: false}},
	}
	mods := []framework.Mod{
		{Key: "a", Enabled: true, Manifest: worldMaps},
		{Key: "b", Enabled: false, Manifest: earthyMaps},
	}
	if got := cleanupHints(mods, nil, nil); len(got) != 0 {
		t.Fatalf("cleanupHints() = %#v, want none", got)
	}
}

// An implicit mod nothing switched on needs is offered for removal; one a switched-on mod requires or lists as optional,
// one switched-on content names in a condition, an explicit one, and an implicit content pack are not.
func TestCleanupHintsFindsOrphanedImplicitMods(t *testing.T) {
	lib := func(id string, implicit bool) framework.Mod {
		m := framework.Mod{Key: id, Enabled: true, Implicit: implicit}
		m.Name, m.UniqueID = id, id
		return m
	}
	user := lib("Me.User", false)
	user.Dependencies = []mod.Dependency{{UniqueID: "Lib.Used", Required: true}, {UniqueID: "Lib.Optional"}}
	pack := lib("Lib.Tiles", true)
	pack.ContentPackFor = "Pathoschild.ContentPatcher"
	got := cleanupHints([]framework.Mod{user, lib("Lib.Used", true), lib("Lib.Optional", true), lib("Lib.Named", true), lib("Lib.Orphan", true), lib("Me.Chosen", false), pack}, nil, []string{"smapi:lib.named"})
	if len(got) != 1 || got[0].Key != "Lib.Orphan" || got[0].Reason != orphanReason {
		t.Fatalf("cleanup = %+v", got)
	}
}

// A switched-on mod that needs a newer game than the install is a row; one the install is new enough for, or with no
// known install version, is not.
func TestNeedsNewerGame(t *testing.T) {
	mod := func(id, needs string) framework.Mod {
		m := framework.Mod{Key: id, Enabled: true}
		m.Name, m.UniqueID, m.MinimumGameVersion = id, id, needs
		return m
	}
	mods := []framework.Mod{mod("too.new", "1.21.0"), mod("fine", "1.20.0"), mod("none", "")}
	if got := needsNewerGame("1.20.12", mods); len(got) != 1 || got[0].Key != "too.new" || got[0].NeedsGame != "1.21.0" {
		t.Fatalf("rows = %+v", got)
	}
	if got := needsNewerGame("", mods); len(got) != 0 {
		t.Fatalf("an unknown install version made rows: %+v", got)
	}
}
