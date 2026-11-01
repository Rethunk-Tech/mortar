package control

import (
	"fmt"
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
// most of them use (SMAPI's when there are none). A bare id that two mods of different formats match is an error,
// since picking one would act on a mod the user may not have meant.
func typedID(known []mod.ID, typed string) (mod.ID, error) {
	counts := map[string]int{}
	common := mod.FormatSMAPI
	var found mod.ID
	for _, id := range known {
		if idMatches(typed, id) {
			if found != "" && !mod.Equal(found, id) {
				return "", fmt.Errorf("ambiguous id %s: give format:id (%s or %s)", strings.TrimSpace(typed), found, id)
			}
			if found == "" {
				found = id
			}
		}
		if counts[id.Format()]++; counts[id.Format()] > counts[common] {
			common = id.Format()
		}
	}
	if found != "" {
		return found, nil
	}
	return mod.Parse(typed, common), nil
}

func typedIDs(known []mod.ID, typed []string) ([]mod.ID, error) {
	out := make([]mod.ID, len(typed))
	for i, t := range typed {
		id, err := typedID(known, t)
		if err != nil {
			return nil, err
		}
		out[i] = id
	}
	return out, nil
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
