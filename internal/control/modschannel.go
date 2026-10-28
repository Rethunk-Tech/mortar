package control

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (s *Services) modsChannel(p Params, prof profile.Profile, id string) (any, error) {
	if p.Value == "" {
		return nil, fmt.Errorf("mods channel needs main, optional, or beta")
	}
	return s.setEach(p.Game, id, prof, p.IDs, func(k string) error {
		_, err := s.Profiles.SetUpdateChannel(p.Game, id, k, p.Value)
		return err
	})
}

// setEach applies set to the entry key of each mod ids names, then returns the refreshed mod rows.
func (s *Services) setEach(game, id string, prof profile.Profile, ids []string, set func(key string) error) (any, error) {
	keys, err := keysFor(prof, ids)
	if err != nil {
		return nil, err
	}
	return s.changed(game, func() (any, error) {
		for _, k := range keys {
			if err := set(k); err != nil {
				return nil, err
			}
		}
		return modRows(s.reload(game, id, prof)), nil
	})
}
