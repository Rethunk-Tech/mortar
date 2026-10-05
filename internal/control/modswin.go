package control

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (s *Services) modsWin(p Params, prof profile.Profile, id string) (any, error) {
	if len(p.IDs) < 2 {
		return nil, fmt.Errorf("mods win needs a winner and a loser")
	}
	keys, err := keysFor(prof, p.IDs[:1])
	if err != nil {
		return nil, err
	}
	return s.changed(p.Game, func() (any, error) {
		return s.Profiles.SetWinner(p.Game, id, keys[0], typedID(p.IDs[1]), !p.Remove)
	})
}
