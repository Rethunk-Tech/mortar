package cp

import (
	"slices"
)

// markLoadAfterWinner marks the conflict decided when every clashing pair has a settled order: one of the two loads
// after the other, or a third pack that clashes with both loads after both and so overwrites them. SMAPI loads a mod
// after everything it depends on, so a dependency settles a pair as surely as a chosen win.
func markLoadAfterWinner(c *Conflict, hits []packHit) {
	byID := byIDOf(hits)
	winners := map[string]packHit{}
	for _, a := range hits {
		for _, rival := range a.Rivals {
			b := byID[rival.Fold()]
			switch {
			case settledAfter(a, b, byID):
				winners[a.ID.Fold()] = a
			case settledAfter(b, a, byID):
				winners[b.ID.Fold()] = b
			case !overwritten(a, b, hits, byID):
				return
			}
		}
	}
	if len(winners) == 0 {
		return
	}
	c.Cosmetic = true
	c.WinnerID, c.WinnerName, c.WinnerKind = "", "", WinnerPerEntry
	for _, w := range winners {
		if allAfter(w, winners, byID) {
			c.WinnerID, c.WinnerName, c.WinnerKind = (w.ID), w.Name, WinnerChosen
			return
		}
	}
}

// allAfter reports whether w loads after every other winner, which makes it the one whose edits stand.
func allAfter(w packHit, winners, byID map[string]packHit) bool {
	for id, o := range winners {
		if id != w.ID.Fold() && !settledAfter(w, o, byID) {
			return false
		}
	}
	return true
}

// overwritten reports a pack that clashes with both a and b on the very keys a and b contest and loads after both.
func overwritten(a, b packHit, hits []packHit, byID map[string]packHit) bool {
	contested := contestedShapes(a, b)
	for _, w := range hits {
		if settledAfter(w, a, byID) && settledAfter(w, b, byID) && rivals(w, a) && rivals(w, b) && touches(w, contested) {
			return true
		}
	}
	return false
}

// contestedShapes is the part of a's edits that b's edits also reach.
func contestedShapes(a, b packHit) []Shape {
	var out []Shape
	for _, x := range a.Edits {
		for _, y := range b.Edits {
			for _, s := range x.Shapes {
				if slices.ContainsFunc(y.Shapes, s.overlaps) {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func touches(w packHit, shapes []Shape) bool {
	return slices.ContainsFunc(w.Edits, func(e Patch) bool { return shapesOverlap(e.Shapes, shapes) })
}

func rivals(a, b packHit) bool {
	return slices.ContainsFunc(a.Rivals, func(id ModID) bool { return Equal(id, b.ID) })
}

// settledAfter reports whether SMAPI loads a after b: through a chosen win or a dependency, directly or through other
// packs in the same conflict.
func settledAfter(a, b packHit, byID map[string]packHit) bool {
	target := b.ID.Fold()
	seen := map[string]bool{a.ID.Fold(): true}
	next := []packHit{a}
	for len(next) > 0 {
		h := next[len(next)-1]
		next = next[:len(next)-1]
		for _, edges := range []map[string]bool{h.LoadAfter, h.Dependencies} {
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
		out[h.ID.Fold()] = h
	}
	return out
}
