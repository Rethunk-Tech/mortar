package problems

import (
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func markLoadAfterWinner(c *AssetConflict, hits []packHit) {
	winner, ok := loadAfterWinner(hits)
	if !ok {
		return
	}
	c.WinnerID = winner.id
	c.WinnerName = winner.name + " wins"
	c.Cosmetic = true
}

func loadAfterWinner(hits []packHit) (packHit, bool) {
	var winner packHit
	found := 0
	for i, a := range hits {
		for j, b := range hits {
			if i == j || !a.loadAfter[strings.ToLower(b.id)] {
				continue
			}
			if found > 0 && !profile.SameID(winner.id, a.id) {
				return packHit{}, false
			}
			winner, found = a, found+1
		}
	}
	if found == 0 {
		return packHit{}, false
	}
	return winner, true
}
