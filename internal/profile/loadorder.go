package profile

import "github.com/Rethunk-AI/mortar/internal/loadorder"

// LoadOrder lists the profile's enabled mods in the order SMAPI loads them.
func (s *Service) LoadOrder(game, id string) ([]loadorder.Row, error) {
	mods, err := s.store.UserMods(game, id)
	if err != nil {
		return nil, err
	}
	in := make([]loadorder.Mod, 0, len(mods))
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		in = append(in, loadorder.Mod{
			UniqueID:       m.UniqueID,
			Name:           m.Name,
			Needs:          m.Needs,
			Optional:       m.Optional,
			ContentPackFor: m.ContentPackFor,
		})
	}
	return loadorder.Resolve(in), nil
}
