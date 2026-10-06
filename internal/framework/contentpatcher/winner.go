package contentpatcher

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// decidedPerEntry is the WinnerName of an edit conflict the user settled with more than one winner.
const decidedPerEntry = "decided per entry"

// markLoadAfterWinner marks the conflict decided when every clashing pair has a settled order: one of the two loads
// after the other, or a third pack that clashes with both loads after both and so overwrites them. SMAPI loads a mod
// after everything it depends on, so a dependency settles a pair as surely as a chosen win.
func markLoadAfterWinner(c *framework.AssetConflict, hits []packHit) {
	byID := byIDOf(hits)
	winners := map[string]packHit{}
	for _, a := range hits {
		for _, rival := range a.rivals {
			b := byID[rival.Fold()]
			switch {
			case settledAfter(a, b, byID):
				winners[a.id.Fold()] = a
			case settledAfter(b, a, byID):
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
	c.WinnerID, c.WinnerName = "", decidedPerEntry
	for _, w := range winners {
		if allAfter(w, winners, byID) {
			c.WinnerID, c.WinnerName = w.id, w.name+" wins"
			return
		}
	}
}

// allAfter reports whether w loads after every other winner, which makes it the one whose edits stand.
func allAfter(w packHit, winners, byID map[string]packHit) bool {
	for id, o := range winners {
		if id != w.id.Fold() && !settledAfter(w, o, byID) {
			return false
		}
	}
	return true
}

// overwritten reports a pack that clashes with both a and b and loads after both.
// ponytail: compares whole packs, not the keys each pair fights over; a winner that clashes with a and b on other
// keys than theirs still counts. Track rivals per key if that shows up.
func overwritten(a, b packHit, hits []packHit) bool {
	for _, w := range hits {
		if settledAfter(w, a, byIDOf(hits)) && settledAfter(w, b, byIDOf(hits)) && rivals(w, a) && rivals(w, b) {
			return true
		}
	}
	return false
}

func rivals(a, b packHit) bool {
	return slices.ContainsFunc(a.rivals, func(id mod.ID) bool { return mod.Equal(id, b.id) })
}

// settledAfter reports whether SMAPI loads a after b: through a chosen win or a dependency, directly or through other
// packs in the same conflict.
func settledAfter(a, b packHit, byID map[string]packHit) bool {
	target := b.id.Fold()
	seen := map[string]bool{a.id.Fold(): true}
	next := []packHit{a}
	for len(next) > 0 {
		h := next[len(next)-1]
		next = next[:len(next)-1]
		for _, edges := range []map[string]bool{h.loadAfter, h.dependencies} {
			for id := range edges {
				if id == target {
					return true
				}
				if p, ok := byID[id]; ok && !seen[id] {
					seen[id] = true
					next = append(next, p)
				}
			}
		}
	}
	return false
}

func byIDOf(hits []packHit) map[string]packHit {
	out := make(map[string]packHit, len(hits))
	for _, h := range hits {
		out[h.id.Fold()] = h
	}
	return out
}
