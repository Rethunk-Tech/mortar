package contentpatcher

import (
	"os"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
)

func TestMain(m *testing.M) {
	loadCacheRoot()
	os.Exit(m.Run())
}

// check analyzes mods as one profile.
func check(mods []framework.Mod) framework.Findings {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m framework.Mod) bool { return !m.Enabled })
	return Driver{}.Analyze(framework.Input{Enabled: enabled, All: mods})
}

// assetConflicts is the conflicts of mods analyzed as one profile.
func assetConflicts(mods []cp.Mod) []cp.Conflict {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m cp.Mod) bool { return !m.Enabled })
	return cp.Analyze(cp.Request{Enabled: enabled, All: mods}).Conflicts
}
