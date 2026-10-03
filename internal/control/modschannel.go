package control

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (s *Services) modsChannel(p Params, prof profile.Profile, id string) (any, error) {
	if p.Value == "" {
		return nil, fmt.Errorf("mods channel needs main, optional, or beta")
	}
	keys, err := keysFor(prof, p.UniqueIDs)
	if err != nil {
		return nil, err
	}
	return s.changed(p.Game, func() (any, error) {
		for _, k := range keys {
			if _, err := s.Profiles.SetUpdateChannel(p.Game, id, k, p.Value); err != nil {
				return nil, err
			}
		}
		return modRows(s.reload(p.Game, id, prof)), nil
	})
}
