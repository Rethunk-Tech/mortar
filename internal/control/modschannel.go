package control

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (s *Services) modsChannel(p Params, prof profile.Profile, id string) (any, error) {
	if p.Value == "" {
		return nil, fmt.Errorf("mods channel needs main, optional, or beta")
	}
	keys, err := keysFor(prof, p.UniqueIDs)
	if err != nil {
		return nil, err
	}
	return s.setEach(p.Game, id, prof, keys, func(k string) error {
		_, err := s.Profiles.SetUpdateChannel(p.Game, id, k, p.Value)
		return err
	})
}

// setEach applies set to every key, then returns the refreshed mod rows.
func (s *Services) setEach(game, id string, prof profile.Profile, keys []string, set func(key string) error) (any, error) {
	return s.changed(game, func() (any, error) {
		for _, k := range keys {
			if err := set(k); err != nil {
				return nil, err
			}
		}
		return modRows(s.reload(game, id, prof)), nil
	})
}
