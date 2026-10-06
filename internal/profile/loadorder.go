package profile

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// LoadOrder lists the profile's enabled mods in the order SMAPI loads them.
func (s *Service) LoadOrder(game, id string) ([]loadorder.Row, error) {
	mods, err := s.store.UserMods(game, id)
	if err != nil {
		return nil, err
	}
	return loadorder.Resolve(loadOrderInput(mods)), nil
}

// loadOrderInput takes the enabled mods. A mod's Needs lists every dependency and Optional the ones its manifest does
// not require, while the load order's Needs holds only the required ones, so an absent optional mod is never missing.
func loadOrderInput(mods []Mod) []loadorder.Mod {
	in := make([]loadorder.Mod, 0, len(mods))
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		required := slices.DeleteFunc(slices.Clone(m.Needs), func(id mod.ID) bool {
			return slices.ContainsFunc(m.Optional, func(o mod.ID) bool { return o.Fold() == id.Fold() })
		})
		in = append(in, loadorder.Mod{
			ID:             m.ID,
			Name:           m.Name,
			Needs:          required,
			Optional:       m.Optional,
			ContentPackFor: m.ContentPackFor,
		})
	}
	return in
}
