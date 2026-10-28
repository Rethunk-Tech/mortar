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

	got := cleanupHints(mods, nil, nil, nil, nil)
	if len(got) != 2 || got[0].Key != "content-framework" || got[1].Key != "spacecore" {
		t.Fatalf("cleanupHints() = %#v, want both unused frameworks", got)
	}
	// A framework the player builds in a Develop project stays, even when only switched-off mods need it.
	mods[0].SourceFolder = "/projects/SpaceCore/artifacts/bin/SpaceCore/debug"
	if got := cleanupHints(mods, nil, nil, nil, nil); len(got) != 1 || got[0].Key != "content-framework" {
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
	if got := cleanupHints(mods, nil, nil, nil, nil); len(got) != 0 {
		t.Fatalf("cleanupHints() = %#v, want none", got)
	}
}

// An implicit mod nothing switched on needs is offered for removal; one a switched-on mod requires or lists as optional,
// one switched-on content names in a condition, an explicit one, an implicit content pack and one the player develops
// (a build of their folder, or a package they publish) are not.
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
	built := lib("Me.Built", true)
	built.SourceFolder = "/projects/Built/bin/debug"
	published := lib("Me.Published", true)
	published.SourceName = "Me-Published"
	got := cleanupHints([]framework.Mod{user, lib("Lib.Used", true), lib("Lib.Optional", true), lib("Lib.Named", true), lib("Lib.Orphan", true), lib("Me.Chosen", false), pack, built, published}, nil, []string{"smapi:lib.named"}, []string{"me-published"}, nil)
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

func TestADropInReplacementMeetsTheDependency(t *testing.T) {
	ts := func(id string, enabled bool, needs ...string) framework.Mod {
		m := framework.Mod{Enabled: enabled}
		m.UniqueID, m.Format = id, mod.FormatThunderstore
		for _, n := range needs {
			m.Dependencies = append(m.Dependencies, mod.Dependency{UniqueID: n, Format: mod.FormatThunderstore, Required: true})
		}
		return m
	}
	pack := ts("Ns-Pack", true, "Old-Loader")
	dropIns := map[string]string{"old-loader": "Mortar-Loader"}
	for name, c := range map[string]struct {
		mods []framework.Mod
		want int
	}{
		"replacement enabled":   {[]framework.Mod{pack, ts("Mortar-Loader", true)}, 0},
		"replacement disabled":  {[]framework.Mod{pack, ts("Mortar-Loader", false)}, 1},
		"replacement not there": {[]framework.Mod{pack}, 1},
		"original disabled":     {[]framework.Mod{pack, ts("Old-Loader", false), ts("Mortar-Loader", true)}, 0},
	} {
		if got := missingDeps("", dropIns, []framework.Mod{pack}, indexByID(c.mods)); len(got) != c.want {
			t.Errorf("%s: missing = %+v", name, got)
		}
	}
}

// What a framework finds unused is not offered either when the mod is one the player develops.
func TestCleanupHintsKeepsDevelopedModsFrameworksFound(t *testing.T) {
	built := framework.Mod{Key: "built", Enabled: true, SourceFolder: "/projects/Tiles/bin/debug"}
	built.Name, built.UniqueID = "Tiles", "Me.Tiles"
	other := framework.Mod{Key: "other", Enabled: true}
	other.Name, other.UniqueID = "Other", "Someone.Other"
	found := []framework.Cleanup{{Key: "built", ID: built.ModID(), Name: "Tiles"}, {Key: "other", ID: other.ModID(), Name: "Other"}}
	got := cleanupHints([]framework.Mod{built, other}, found, nil, nil, nil)
	if len(got) != 1 || got[0].Key != "other" {
		t.Fatalf("cleanup = %+v", got)
	}
}

func TestAThunderstoreDependencyIsFetchedFromItsPackagePage(t *testing.T) {
	r := thunderstoreRef("lethal-company", mod.ID("thunderstore:femboytv-LethalPosters:1.2"))
	if r == nil || r.Package != "femboytv-LethalPosters" || r.URL != "https://thunderstore.io/c/lethal-company/p/femboytv/LethalPosters/" {
		t.Fatalf("ref = %+v", r)
	}
	if thunderstoreRef("", mod.ID("thunderstore:a-b")) != nil || thunderstoreRef("lethal-company", mod.ID("smapi:Pathoschild.ContentPatcher")) != nil {
		t.Fatal("a ref with no community or for another format")
	}
}

func TestCleanupHintsKeepsTheDropInReplacementOfANeededPackage(t *testing.T) {
	user := framework.Mod{Key: "user", Enabled: true}
	user.Format, user.UniqueID = mod.FormatThunderstore, "Ns-User"
	user.Dependencies = []mod.Dependency{{Format: mod.FormatThunderstore, UniqueID: "Ns-Old", Required: true}}
	repl := framework.Mod{Key: "new", Enabled: true, Implicit: true}
	repl.Format, repl.UniqueID = mod.FormatThunderstore, "Ns-New"
	mods := []framework.Mod{user, repl}
	if got := cleanupHints(mods, nil, nil, nil, nil); len(got) != 1 {
		t.Fatalf("without the drop-in the replacement is an orphan: %+v", got)
	}
	if got := cleanupHints(mods, nil, nil, nil, map[string]string{"ns-old": "ns-new"}); len(got) != 0 {
		t.Fatalf("cleanup = %+v", got)
	}
}
