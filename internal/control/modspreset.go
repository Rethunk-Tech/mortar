package control

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (s *Services) modsPreset(p Params, id string, prof profile.Profile) (any, error) {
	if len(p.IDs) == 0 {
		return nil, fmt.Errorf("mods preset needs a mod")
	}
	refs, err := refsFor(prof, p.IDs[:1])
	if err != nil {
		return nil, err
	}
	ref := refs[0]
	switch p.Sub {
	case "list":
		return s.Profiles.ListConfigPresets(p.Game, ref.ID)
	case "save":
		if p.Name == "" {
			return nil, fmt.Errorf("mods preset save needs a name")
		}
		raw, err := s.Profiles.ReadConfig(p.Game, id, ref.Key, ref.ID)
		if err != nil {
			return nil, err
		}
		if err := s.Profiles.SaveConfigPreset(p.Game, ref.ID, p.Name, raw); err != nil {
			return nil, err
		}
		return s.Profiles.ListConfigPresets(p.Game, ref.ID)
	case "apply":
		if p.Name == "" {
			return nil, fmt.Errorf("mods preset apply needs a name")
		}
		return s.changed(p.Game, func() (any, error) {
			return nil, s.Profiles.ApplyConfigPreset(p.Game, id, ref.Key, ref.ID, p.Name)
		})
	case "delete":
		if p.Name == "" {
			return nil, fmt.Errorf("mods preset delete needs a name")
		}
		if err := s.Profiles.DeleteConfigPreset(p.Game, ref.ID, p.Name); err != nil {
			return nil, err
		}
		return s.Profiles.ListConfigPresets(p.Game, ref.ID)
	default:
		return nil, fmt.Errorf("mods preset needs list, save, apply, or delete")
	}
}
