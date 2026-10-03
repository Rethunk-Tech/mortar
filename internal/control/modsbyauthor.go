package control

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (s *Services) modsByAuthor(p Params) ([]profile.AuthorMod, error) {
	if p.Query == "" {
		return nil, fmt.Errorf("mods by-author needs an author name")
	}
	return s.Profiles.ModsByAuthor(p.Game, p.Query)
}
