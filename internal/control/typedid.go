package control

import (
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// idMatches reports whether text a user typed names id: a full "format:local" id by the format's rule, a bare one by
// the local part alone, so a Thunderstore or BepInEx mod can be named as plainly as a SMAPI one.
func idMatches(typed string, id mod.ID) bool {
	typed = strings.TrimSpace(typed)
	if strings.Contains(typed, ":") {
		return mod.Equal(mod.ID(typed), id)
	}
	return mod.Equal(mod.NewID(id.Format(), typed), id)
}

// typedID reads an id a user typed against the mods it may name: the one it matches, else the bare id in the format
// most of them use (SMAPI's when there are none).
func typedID(known []mod.ID, typed string) mod.ID {
	counts := map[string]int{}
	common := mod.FormatSMAPI
	for _, id := range known {
		if idMatches(typed, id) {
			return id
		}
		if counts[id.Format()]++; counts[id.Format()] > counts[common] {
			common = id.Format()
		}
	}
	return mod.Parse(typed, common)
}

func typedIDs(known []mod.ID, typed []string) []mod.ID {
	out := make([]mod.ID, len(typed))
	for i, t := range typed {
		out[i] = typedID(known, t)
	}
	return out
}

// modIDs is every mod id a profile holds, for typedID.
func modIDs(p profile.Profile) []mod.ID {
	var out []mod.ID
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			out = append(out, m.ID)
		}
	}
	return out
}
