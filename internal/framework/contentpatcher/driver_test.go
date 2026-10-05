package contentpatcher

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

// check analyzes mods as one profile.
func check(mods []framework.Mod) framework.Findings {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m framework.Mod) bool { return !m.Enabled })
	return Driver{}.Analyze(framework.Input{Enabled: enabled, All: mods})
}
