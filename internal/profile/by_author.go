package profile

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// AuthorMod is one mod installed under an author name somewhere in the game.
type AuthorMod struct {
	ID       mod.ID         `json:"id"`
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
					ID:          m.ID,
					Name:        m.Name,
					Version:     m.Version,
					Enabled:     e.Enabled(m.ID),
				}
				existing := byID[m.ID.Fold()]
				if existing == nil {
					byID[m.ID.Fold()] = &AuthorMod{
						ID:       m.ID,
						Name:     m.Name,
						Profiles: []ModInProfile{row},
					}
					continue
				}
				existing.Name = cmp.Or(existing.Name, m.Name)
				existing.Profiles = append(existing.Profiles, row)
			}
		}
	}
	out := make([]AuthorMod, 0, len(byID))
	for _, im := range byID {
		slices.SortFunc(im.Profiles, func(a, b ModInProfile) int {
			if n := strings.Compare(strings.ToLower(a.ProfileName), strings.ToLower(b.ProfileName)); n != 0 {
				return n
			}
			return strings.Compare(a.ID.Fold(), b.ID.Fold())
		})
		out = append(out, *im)
	}
	slices.SortFunc(out, func(a, b AuthorMod) int {
		return compareNameThenID(a.Name, a.ID, b.Name, b.ID)
	})
	return out, nil
}
