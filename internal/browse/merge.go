package browse

import (
	"regexp"
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

// minTitleLen keeps a very short name from tying unrelated mods.
const minTitleLen = 4

var (
	githubLink      = regexp.MustCompile(`(?i)github\.com/([\w.-]+/[\w.-]+)`)
	thunderstoreRef = regexp.MustCompile(`(?i)thunderstore\.io/c/[\w-]+/p/([\w]+)/([\w]+)`)
)

// keys are what ties two hits to one mod across sources: the game's loader, the GitHub repository or Thunderstore
// package its site or summary links, the exact name and author, or the exact name alone. A name alone can tie two
// different mods, but never two hits of one source, and a site's author is often the uploader.
func keys(it Item) []string {
	var out []string
	if it.Loader {
		out = append(out, "loader")
	}
	if it.Repo != "" {
		out = append(out, "repo:"+strings.ToLower(it.Repo))
	}
	if m := githubLink.FindStringSubmatch(it.Summary); m != nil {
		out = append(out, "repo:"+strings.ToLower(strings.TrimSuffix(m[1], ".git")))
	}
	if m := thunderstoreRef.FindStringSubmatch(it.Summary); m != nil {
		out = append(out, "ts:"+strings.ToLower(m[1]+"-"+m[2]))
	}
	if it.Source == "thunderstore" && it.ID != "" {
		out = append(out, "ts:"+strings.ToLower(it.ID))
	}
	n := foldKey(it.Name)
	if a := foldKey(it.Author); n != "" && a != "" {
		out = append(out, "name:"+n+"|"+a)
	}
	if len(n) >= minTitleLen {
		out = append(out, "title:"+n)
	}
	return out
}

// mergeSame folds hits of the same mod from different sources into one card, in the position of the group's first
// hit. The card is the hit of the earliest source in order (the game's catalog order), and the rest are its Alts,
// each with its own Installed, Obsolete, Broken and Loader. A card with a loader hit is a loader card.
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
			card.Loader = card.Loader || m.Loader
			if i != best {
				card.Alts = append(card.Alts, source.Alt{
					Source: m.Source, ID: m.ID, URL: m.URL, Installed: m.Installed,
					Obsolete: m.Obsolete, Broken: m.Broken, Loader: m.Loader,
				})
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
