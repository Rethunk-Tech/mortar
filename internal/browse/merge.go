package browse

import (
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Both need the "//" of a link right before the host, so a host that only ends in the name (evilgithub.com) is not it.
var (
	githubLink      = regexp.MustCompile(`(?i)//(?:www\.)?github\.com/([\w.-]+/[\w.-]+)`)
	thunderstoreRef = regexp.MustCompile(`(?i)//(?:www\.)?thunderstore\.io/c/[\w-]+/p/([\w]+)/([\w]+)`)
)

// identityOf resolves a hit to its package identity (a SMAPI UniqueID or a loader plugin id), "" when none is known.
type identityOf func(Item) string

// keys are what ties two hits to one mod across sources, and only references and identities do: the game's loader,
// the package identity both hits resolve to, the GitHub repository a hit links, or the Thunderstore package its
// summary points at. A name never ties two hits: different mods share names, and a site's author is often the
// uploader.
func keys(it Item, ident identityOf) []string {
	var out []string
	if it.Loader {
		out = append(out, "loader")
	}
	if ident != nil {
		if id := ident(it); id != "" {
			out = append(out, "id:"+id)
		}
	}
	if it.Repo != "" {
		out = append(out, "repo:"+strings.ToLower(it.Repo))
	}
	if m := githubLink.FindStringSubmatch(it.Summary); m != nil {
		// A link that ends a sentence captures its full stop, which no repository name ends in.
		out = append(out, "repo:"+strings.ToLower(strings.TrimSuffix(strings.TrimRight(m[1], "."), ".git")))
	}
	if m := thunderstoreRef.FindStringSubmatch(it.Summary); m != nil {
		out = append(out, "ts:"+strings.ToLower(m[1]+"-"+m[2]))
	}
	if it.Source == "thunderstore" && it.ID != "" {
		out = append(out, "ts:"+strings.ToLower(it.ID))
	}
	return out
}

// mergeSame folds hits of the same mod from different sources into one card, in the position of the group's first
// hit. The card is the hit of the earliest source in order (the game's catalog order), and the rest are its Alts,
// each with its own Installed, Obsolete, Broken, Loader and counts. A card with a loader hit is a loader card.
// Two hits of one source are never merged: a source already lists a mod once.
func mergeSame(items []Item, order []string, ident identityOf) []Item {
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
		for _, k := range keys(it, ident) {
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
		for _, k := range keys(it, ident) {
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
					Endorsements: m.Endorsements, Stars: m.Stars, Downloads: m.Downloads,
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
