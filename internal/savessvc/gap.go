package savessvc

import (
	"cmp"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

// GapMod is a mod a save was last played with. Version is the recorded one; Have is the profile's version of an
// enabled mod that is older, and empty otherwise.
type GapMod struct {
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Have     string `json:"have"`
}

// Gap is what the profile lacks of the mods a save was last played with: absent, switched off, or enabled at an
// older version.
type Gap struct {
	Missing      []GapMod `json:"missing"`
	Disabled     []GapMod `json:"disabled"`
	VersionOlder []GapMod `json:"versionOlder"`
}

// SaveGap compares the mods the save was last played with against the profile. A save never played through Mortar
// has no recorded list and no gap.
func (s *Service) SaveGap(game, profileID, save string) (Gap, error) {
	gap := Gap{Missing: []GapMod{}, Disabled: []GapMod{}, VersionOlder: []GapMod{}}
	if s.last == nil || save == "" {
		return gap, nil
	}
	rec, ok, err := s.last.Get(game, save)
	if err != nil || !ok {
		return gap, err
	}
	mods, err := s.profiles.Mods(game, profileID)
	if err != nil {
		return gap, err
	}
	present, enabled := haveMaps(mods)
	recorded := map[string]PlayedMod{}
	for _, m := range rec.Mods {
		recorded[strings.ToLower(m.UniqueID)] = m
	}
	for _, l := range MissingFrom(rec.Mods, present, enabled) {
		m := GapMod{UniqueID: l.UniqueID, Name: l.Name, Version: recorded[strings.ToLower(l.UniqueID)].Version}
		if l.Disabled {
			gap.Disabled = append(gap.Disabled, m)
		} else {
			gap.Missing = append(gap.Missing, m)
		}
	}
	gap.VersionOlder = olderThanRecorded(sortRecorded(rec.Mods), mods)
	return gap, nil
}

func olderThanRecorded(recorded []PlayedMod, mods []profile.Mod) []GapMod {
	have := map[string]string{}
	for _, m := range mods {
		if m.Enabled {
			have[strings.ToLower(m.UniqueID)] = m.Version
		}
	}
	out := []GapMod{}
	for _, r := range recorded {
		if v, ok := have[strings.ToLower(r.UniqueID)]; ok && meta.Newer(r.Version, v) {
			out = append(out, GapMod{UniqueID: r.UniqueID, Name: cmp.Or(r.Name, r.UniqueID), Version: r.Version, Have: v})
		}
	}
	return out
}
