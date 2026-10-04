package nexussvc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// TrackedMissing lists tracked mods for the game's Nexus domain that no Nexus entry in the profile uses.
func (s *Service) TrackedMissing(ctx context.Context, gameID, profileID string) ([]nexus.TrackedMod, error) {
	if _, err := s.keyed(); err != nil {
		return nil, err
	}
	info, ok := components.BundledGame(gameID)
	if !ok || info.Nexus.Domain == "" {
		return nil, fmt.Errorf("game %q has no Nexus domain", gameID)
	}
	if s.Profiles == nil {
		return nil, errors.New("profiles are unavailable")
	}
	p, err := s.profileByID(gameID, profileID)
	if err != nil {
		return nil, err
	}
	installed := profileNexusModIDs(p)
	mods, err := s.TrackedMods(ctx)
	if err != nil {
		return nil, err
	}
	var out []nexus.TrackedMod
	for _, mod := range mods {
		if !strings.EqualFold(mod.DomainName, info.Nexus.Domain) {
			continue
		}
		if installed[mod.ModID] {
			continue
		}
		out = append(out, mod)
	}
	return out, nil
}

func (s *Service) profileByID(gameID, profileID string) (profile.Profile, error) {
	profiles, err := s.Profiles.List(gameID)
	if err != nil {
		return profile.Profile{}, err
	}
	for _, p := range profiles {
		if p.ID == profileID {
			if p.Error != "" {
				return profile.Profile{}, fmt.Errorf("profile %q is damaged", profileID)
			}
			return p, nil
		}
	}
	return profile.Profile{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("profile %q not found", profileID))
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
