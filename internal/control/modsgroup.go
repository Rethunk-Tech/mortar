package control

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (s *Services) modsGroup(p Params, prof profile.Profile, id string) (any, error) {
	switch p.Sub {
	case "list":
		if prof.Groups == nil {
			return []profile.Group{}, nil
		}
		return prof.Groups, nil
	case "create":
		return s.changed(p.Game, func() (any, error) { return s.Profiles.CreateGroup(p.Game, id, p.Name) })
	case "delete":
		return s.changed(p.Game, func() (any, error) { return s.Profiles.DeleteGroup(p.Game, id, p.Name) })
	case "add", "remove":
		if len(p.IDs) == 0 {
			return nil, fmt.Errorf("mods group %s needs a mod id", p.Sub)
		}
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		key := keys[0]
		return s.changed(p.Game, func() (any, error) {
			if p.Sub == "add" {
				return s.Profiles.AddToGroup(p.Game, id, p.Name, key)
			}
			return s.Profiles.RemoveFromGroup(p.Game, id, p.Name, key)
		})
	case "on", "off":
		return s.changed(p.Game, func() (any, error) {
			return s.Profiles.SetGroupEnabled(p.Game, id, p.Name, p.Sub == "on")
		})
	default:
		return nil, fmt.Errorf("mods group needs list, create, delete, add, remove, on, or off")
	}
}
