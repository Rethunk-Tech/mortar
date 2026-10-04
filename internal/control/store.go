package control

import (
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

func (s *Services) storeReport(p Params) (any, error) {
	if s.Data == nil {
		return nil, errors.New("data is unavailable")
	}
	rep, err := s.Data.Report()
	if err != nil {
		return nil, err
	}
	if p.Game == "" {
		return rep, nil
	}
	out := store.Report{}
	if g, ok := rep[p.Game]; ok {
		out[p.Game] = g
	}
	return out, nil
}

func (s *Services) storeRemove(p Params) (any, error) {
	if s.Data == nil {
		return nil, errors.New("data is unavailable")
	}
	return nil, s.Data.RemoveItems(p.Game, p.UniqueIDs)
}
