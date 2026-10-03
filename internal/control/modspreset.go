package control

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (s *Services) modsPreset(p Params, id string, prof profile.Profile) (any, error) {
	if len(p.UniqueIDs) == 0 {
		return nil, fmt.Errorf("mods preset needs a mod")
	}
	refs, err := refsFor(prof, p.UniqueIDs[:1])
	if err != nil {
		return nil, err
	}
	ref := refs[0]
	switch p.Sub {
	case "list":
		return s.Profiles.ListConfigPresets(p.Game, ref.UniqueID)
	case "save":
		if p.Name == "" {
			return nil, fmt.Errorf("mods preset save needs a name")
		}
		raw, err := s.Profiles.ReadConfig(p.Game, id, ref.Key, ref.UniqueID)
		if err != nil {
			return nil, err
		}
		if err := s.Profiles.SaveConfigPreset(p.Game, ref.UniqueID, p.Name, raw); err != nil {
			return nil, err
		}
		return s.Profiles.ListConfigPresets(p.Game, ref.UniqueID)
	case "apply":
		if p.Name == "" {
			return nil, fmt.Errorf("mods preset apply needs a name")
		}
		return s.changed(p.Game, func() (any, error) {
			return nil, s.Profiles.ApplyConfigPreset(p.Game, id, ref.Key, ref.UniqueID, p.Name)
		})
	case "delete":
		if p.Name == "" {
			return nil, fmt.Errorf("mods preset delete needs a name")
		}
		if err := s.Profiles.DeleteConfigPreset(p.Game, ref.UniqueID, p.Name); err != nil {
			return nil, err
		}
		return s.Profiles.ListConfigPresets(p.Game, ref.UniqueID)
	default:
		return nil, fmt.Errorf("mods preset needs list, save, apply, or delete")
	}
}
