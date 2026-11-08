package contentpatcher

import (
	"os"
	"slices"
	"testing"

	cp "github.com/Rethunk-Tech/go-contentpatcher"
)

func TestMain(m *testing.M) {
	loadCacheRoot()
	os.Exit(m.Run())
}

// assetConflicts is the conflicts of mods analyzed as one profile.
func assetConflicts(mods []cp.Mod) []cp.Conflict {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m cp.Mod) bool { return !m.Enabled })
	return cp.Analyze(cp.Request{Enabled: enabled, All: mods}).Conflicts
}

// mentionedIDs are the mods the profile's switched-on packs name in a HasMod condition.
func mentionedIDs(mods []cp.Mod, skip bool) []string {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m cp.Mod) bool { return !m.Enabled })
	return cp.Analyze(cp.Request{Enabled: enabled, All: mods, SkipImageOverlap: skip}).Mentioned
}

// A redundant item is worded by Mortar: "shadowed" for a pack another replaces, "bundled" for one a pack ships
// itself.
func TestRedundantKinds(t *testing.T) {
	got := toFrameworkFindings(cp.Findings{Shadowed: []cp.Shadowed{
		{Key: "a", Name: "A"},
		{Key: "b", Name: "B", Bundled: true},
	}}).Redundant
	if len(got) != 2 || got[0].Kind != "shadowed" || got[1].Kind != "bundled" {
		t.Fatalf("redundant = %+v", got)
	}
}

// A switched-on pack names the mods its conditions ask for and the mods whose own assets it changes; a built-in asset
// names none.
func TestMentionedIDs(t *testing.T) {
	pack := tilesheetCPPack(t, "Me.Pack", `{"Changes":[
		{"Action":"EditData","Target":"Other.Mod/ItemQueries","Entries":{}},
		{"Action":"EditData","Target":"Data/Objects","Entries":{},"When":{"HasMod":"Some.Lib"}}]}`)
	pack.Enabled = true
	if got := mentionedIDs([]cp.Mod{pack}, false); !slices.Equal(got, []string{"smapi:other.mod", "smapi:some.lib"}) {
		t.Fatalf("mentioned = %q", got)
	}
}
