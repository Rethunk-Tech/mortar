package nexussvc

import (
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// TrackedMissing lists tracked mods for the game's Nexus domain that no Nexus entry in the profile uses.
func (s *Service) TrackedMissing(ctx context.Context, gameID, profileID string) ([]nexusmods.TrackedMod, error) {
	if _, err := s.keyed(); errors.Is(err, ErrSignedOut) {
		return []nexusmods.TrackedMod{}, nil
	} else if err != nil {
		return nil, err
	}
	t, err := game.NexusTitle(gameID)
	if err != nil {
		return nil, err
	}
	if s.Profiles == nil {
		return nil, errors.New("profiles are unavailable")
	}
	p, err := s.Profiles.Get(gameID, profileID)
	if err != nil {
		return nil, err
	}
	installed := profileNexusModIDs(p)
	mods, err := s.trackedForCard(ctx)
	if err != nil {
		return nil, err
	}
	var out []nexusmods.TrackedMod
	for _, mod := range mods {
		if !strings.EqualFold(mod.DomainName, t.Domain) {
			continue
		}
		if installed[mod.ModID] {
			continue
		}
		out = append(out, mod)
	}
	return out, nil
}

func profileNexusModIDs(p profile.Profile) map[int]bool {
	used := map[int]bool{}
	for _, entry := range p.Entries {
		if entry.Source.Kind == profile.KindNexus && entry.Source.ModID > 0 {
			used[entry.Source.ModID] = true
		}
	}
	return used
}
