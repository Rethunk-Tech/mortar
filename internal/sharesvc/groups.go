package sharesvc

import (
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/share"
)

func (s *Service) applySharedGroups(game, profileID string, groups []share.FileGroup) error {
	if len(groups) == 0 {
		return nil
	}
	p, err := s.find(game, profileID)
	if err != nil {
		return err
	}
	for _, g := range groups {
		keys := share.ResolveGroupKeys(p, g)
		if len(keys) == 0 {
			if _, err := s.d.Profiles.CreateGroup(game, profileID, g.Name); err != nil {
				if !hasGroup(p, g.Name) {
					return err
				}
			}
			continue
		}
		for _, key := range keys {
			next, err := s.d.Profiles.AddToGroup(game, profileID, g.Name, key)
			if err != nil {
				return err
			}
			p = next
		}
	}
	return nil
}

func hasGroup(p profile.Profile, name string) bool {
	return slices.ContainsFunc(p.Groups, func(g profile.Group) bool {
		return strings.EqualFold(g.Name, name)
	})
}
