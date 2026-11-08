package contentpatcher

import (
	"os"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
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
func mentionedIDs(mods []cp.Mod) []string {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m cp.Mod) bool { return !m.Enabled })
	return cp.Analyze(cp.Request{Enabled: enabled, All: mods}).Mentioned
}

// A switched-on pack names the mods its conditions ask for and the mods whose own assets it changes; a built-in asset
// names none.
func TestMentionedIDs(t *testing.T) {
	pack := tilesheetCPPack(t, "Me.Pack", `{"Changes":[
		{"Action":"EditData","Target":"Other.Mod/ItemQueries","Entries":{}},
		{"Action":"EditData","Target":"Data/Objects","Entries":{},"When":{"HasMod":"Some.Lib"}}]}`)
	pack.Enabled = true
	if got := mentionedIDs([]cp.Mod{pack}); !slices.Equal(got, []string{"smapi:other.mod", "smapi:some.lib"}) {
		t.Fatalf("mentioned = %q", got)
	}
}
