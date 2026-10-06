package contentpatcher

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// decidedPerEntry is the WinnerName of an edit conflict the user settled with more than one winner.
const decidedPerEntry = "decided per entry"

// markLoadAfterWinner marks the conflict decided when every clashing pair has an order the user chose: one of the
// two loads after the other, or a third pack that clashes with both loads after both and so overwrites them.
func markLoadAfterWinner(c *framework.AssetConflict, hits []packHit) {
	byID := make(map[string]packHit, len(hits))
	for _, h := range hits {
		byID[h.id.Fold()] = h
	}
	winners := map[string]packHit{}
	for _, a := range hits {
		for _, rival := range a.rivals {
			b := byID[rival.Fold()]
			switch {
			case a.loadAfter[b.id.Fold()]:
				winners[a.id.Fold()] = a
			case b.loadAfter[a.id.Fold()]:
				winners[b.id.Fold()] = b
			case !overwritten(a, b, hits):
				return
			}
		}
	}
	if len(winners) == 0 {
		return
	}
	c.Cosmetic = true
	if len(winners) > 1 {
		c.WinnerID, c.WinnerName = "", decidedPerEntry
		return
	}
	for _, w := range winners {
		c.WinnerID, c.WinnerName = w.id, w.name+" wins"
	}
}

// overwritten reports a pack that clashes with both a and b and loads after both.
// ponytail: compares whole packs, not the keys each pair fights over; a winner that clashes with a and b on other
// keys than theirs still counts. Track rivals per key if that shows up.
func overwritten(a, b packHit, hits []packHit) bool {
	for _, w := range hits {
		if w.loadAfter[a.id.Fold()] && w.loadAfter[b.id.Fold()] && rivals(w, a) && rivals(w, b) {
			return true
		}
	}
	return false
}

func rivals(a, b packHit) bool {
	return slices.ContainsFunc(a.rivals, func(id mod.ID) bool { return mod.Equal(id, b.id) })
}
