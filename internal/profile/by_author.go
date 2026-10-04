package profile

import (
	"slices"
	"strings"
)

// AuthorMod is one mod installed under an author name somewhere in the game.
type AuthorMod struct {
	UniqueID string         `json:"uniqueId"`
	Name     string         `json:"name"`
	Profiles []ModInProfile `json:"profiles"`
}

// ModsByAuthor lists mods whose manifest Author field includes author in any profile of game.
func (s *Store) ModsByAuthor(game, author string) ([]AuthorMod, error) {
	if NormalizeAuthorName(author) == "" {
		return []AuthorMod{}, nil
	}
	all, err := s.listOK(game)
	if err != nil {
		return nil, err
	}
	byID := map[string]*AuthorMod{}
	for _, prof := range all {
		for _, e := range prof.Entries {
			for _, m := range e.Mods {
				if !AuthorFieldIncludes(m.Author, author) {
					continue
				}
				row := ModInProfile{
					ProfileID:   prof.ID,
					ProfileName: prof.Name,
					Key:         e.Key,
					UniqueID:    m.UniqueID,
					Name:        m.Name,
					Version:     m.Version,
					Enabled:     !hasID(e.Disabled, m.UniqueID),
				}
				existing := byID[strings.ToLower(m.UniqueID)]
				if existing == nil {
					byID[strings.ToLower(m.UniqueID)] = &AuthorMod{
						UniqueID: m.UniqueID,
						Name:     m.Name,
						Profiles: []ModInProfile{row},
					}
					continue
				}
				if existing.Name == "" {
					existing.Name = m.Name
				}
				existing.Profiles = append(existing.Profiles, row)
			}
		}
	}
	out := make([]AuthorMod, 0, len(byID))
	for _, mod := range byID {
		slices.SortFunc(mod.Profiles, func(a, b ModInProfile) int {
			if n := strings.Compare(strings.ToLower(a.ProfileName), strings.ToLower(b.ProfileName)); n != 0 {
				return n
			}
			return strings.Compare(strings.ToLower(a.UniqueID), strings.ToLower(b.UniqueID))
		})
		out = append(out, *mod)
	}
	slices.SortFunc(out, func(a, b AuthorMod) int {
		return compareNameThenID(a.Name, a.UniqueID, b.Name, b.UniqueID)
	})
	return out, nil
}
