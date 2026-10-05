package browse

import (
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// foldKey lowercases s and drops everything but letters and digits, so "Cool Mod!" and "cool-mod" are one name.
func foldKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// keys are what ties two hits to one mod: the GitHub repository their site links, else the exact name and author.
func keys(it Item) []string {
	var out []string
	if it.Repo != "" {
		out = append(out, "repo:"+strings.ToLower(it.Repo))
	}
	if n, a := foldKey(it.Name), foldKey(it.Author); n != "" && a != "" {
		out = append(out, "name:"+n+"|"+a)
	}
	return out
}

// mergeSame folds hits of the same mod from different sources into one card, in the position of the group's first
// hit. The card is the hit of the earliest source in order (the game's catalog order), and the rest are its Alts,
// each with its own Installed.
// Two hits of one source are never merged: a source already lists a mod once.
func mergeSame(items []Item, order []string) []Item {
	rank := func(src string) int {
		for i, id := range order {
			if id == src {
				return i
			}
		}
		return len(order)
	}
	groupOf := map[string]int{}
	var groups [][]Item
	for _, it := range items {
		g := -1
		for _, k := range keys(it) {
			if i, ok := groupOf[k]; ok && !hasSource(groups[i], it.Source) {
				g = i
				break
			}
		}
		if g < 0 {
			g = len(groups)
			groups = append(groups, nil)
		}
		groups[g] = append(groups[g], it)
		for _, k := range keys(it) {
			if _, ok := groupOf[k]; !ok {
				groupOf[k] = g
			}
		}
	}
	out := make([]Item, 0, len(groups))
	for _, members := range groups {
		best := 0
		for i, m := range members {
			if rank(m.Source) < rank(members[best].Source) {
				best = i
			}
		}
		card := members[best]
		for i, m := range members {
			if i != best {
				card.Alts = append(card.Alts, source.Alt{Source: m.Source, ID: m.ID, URL: m.URL, Installed: m.Installed})
			}
		}
		out = append(out, card)
	}
	return out
}

func hasSource(members []Item, src string) bool {
	for _, m := range members {
		if m.Source == src {
			return true
		}
	}
	return false
}
