package control

import (
	"context"
	"errors"
)

// farmCall runs one multiplayer-list verb: export lists the profile's mods; check and fix read the host's list from
// Value.
func (s *Services) farmCall(ctx context.Context, method string, p Params) (any, error) {
	if s.Packs == nil {
		return nil, errors.New("multiplayer lists are unavailable")
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	switch method {
	case "pack.farmExport":
		return s.Packs.ExportFarm(p.Game, prof.ID)
	case "pack.farmCheck":
		return s.Packs.CheckFarm(p.Game, prof.ID, p.Value)
	default:
		return s.Packs.FixFarm(ctx, p.Game, prof.ID, p.Value, typedIDs(p.IDs))
	}
}
