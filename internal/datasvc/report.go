package datasvc

import (
	"fmt"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

// Report is unused store items and duplicate groups, per game.
func (s *Service) Report() (store.Report, error) {
	keys, err := s.referenced()
	if err != nil {
		return nil, err
	}
	return s.items.Report(keys)
}

// RemoveItems deletes store folders for game when no profile still names them.
func (s *Service) RemoveItems(game string, keys []string) error {
	held, err := s.referenced()
	if err != nil {
		return err
	}
	for _, k := range keys {
		if !slices.Contains(held[game], k) {
			continue
		}
		name := s.profileUsing(game, k)
		if name == "" {
			return fmt.Errorf("%w: store item %s", errInUse, k)
		}
		return fmt.Errorf("%w: profile %q still uses store item %s", errInUse, name, k)
	}
	refs := make([]store.Ref, 0, len(keys))
	for _, k := range keys {
		refs = append(refs, store.Ref{Game: game, Key: k})
	}
	if err := s.items.Remove(refs); err != nil {
		return err
	}
	s.forgetModUsage()
	return nil
}

func (s *Service) profileUsing(game, key string) string {
	if s.profiles == nil {
		return ""
	}
	list, err := s.profiles.List(game)
	if err != nil {
		return ""
	}
	for _, p := range list {
		for _, e := range p.Entries {
			if e.Key == key {
				return p.Name
			}
		}
	}
	return ""
}
