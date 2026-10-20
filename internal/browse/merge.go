package browse

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// linkToken finds the http(s) links in free text; what each one points at is decided on the parsed URL, never on the
// text around it, so a link that only carries a host in its query (?next=//github.com/a/b) is not a link to it.
var (
	linkToken = regexp.MustCompile(`(?i)https?://[^\s<>"'()\[\]]+`)
	nameChars = regexp.MustCompile(`^[\w.-]+$`)
	wordChars = regexp.MustCompile(`^\w+$`)
)

// pathOnHost is the non-empty path segments of the first link in text whose host is exactly host (ignoring case and a
// leading www.) and whose path has at least `least` segments and passes ok.
func pathOnHost(text, host string, least int, ok func([]string) bool) []string {
	for _, tok := range linkToken.FindAllString(text, -1) {
		u, err := url.Parse(tok)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		if strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.") != host {
			continue
		}
		var segs []string
		for seg := range strings.SplitSeq(u.Path, "/") {
			if seg != "" {
				segs = append(segs, seg)
			}
		}
		if len(segs) >= least && ok(segs) {
			return segs
		}
	}
	return nil
}

// githubRepoIn is the "owner/repo" of the first GitHub repository link in text, or "".
func githubRepoIn(text string) string {
	segs := pathOnHost(text, "github.com", 2, func(s []string) bool {
		return nameChars.MatchString(s[0]) && nameChars.MatchString(s[1])
	})
	if segs == nil {
		return ""
	}
	return segs[0] + "/" + segs[1]
}

// thunderstoreRefIn is "namespace-name" of the first Thunderstore package link (/c/<community>/p/<namespace>/<name>), or "".
func thunderstoreRefIn(text string) string {
	segs := pathOnHost(text, "thunderstore.io", 5, func(s []string) bool {
		return s[0] == "c" && s[2] == "p" && wordChars.MatchString(s[3]) && wordChars.MatchString(s[4])
	})
	if segs == nil {
		return ""
	}
	return segs[3] + "-" + segs[4]
}

// identityOf resolves a hit to its package identity (a SMAPI UniqueID or a loader plugin id), "" when none is known.
type identityOf func(Item) string

// keys are what ties two hits to one mod across sources: the package identity both hits resolve to, the GitHub
// repository a hit links, the Thunderstore package its summary points at, or the same name by the same author. A name
// alone never ties two hits: different mods share names, and a site's author is often only the uploader.
func keys(it Item, ident identityOf, people aliases) []string {
	var out []string
	if name := foldName(it.Name); name != "" {
		for _, a := range people.authors(it.Author) {
			out = append(out, "name:"+name+"|"+a)
		}
	}
	if ident != nil {
		if id := ident(it); id != "" {
			out = append(out, "id:"+id)
		}
	}
	if it.Repo != "" {
		out = append(out, "repo:"+strings.ToLower(it.Repo))
	}
	if repo := githubRepoIn(it.Summary); repo != "" {
		// A link that ends a sentence captures its full stop, which no repository name ends in.
		out = append(out, "repo:"+strings.ToLower(strings.TrimSuffix(strings.TrimRight(repo, "."), ".git")))
	}
	if ref := thunderstoreRefIn(it.Summary); ref != "" {
		out = append(out, "ts:"+strings.ToLower(ref))
	}
	if it.Source == "thunderstore" && it.ID != "" {
		out = append(out, "ts:"+strings.ToLower(it.ID))
	}
	return out
}

// mergeSame folds hits of the same mod from different sources into one card, in the position of the group's first
// hit. The card is the hit of the earliest source in order (the game's catalog order), and the rest are its Alts,
// each with its own Installed, Obsolete, Broken and counts.
// Two hits of one source are never merged: a source already lists a mod once.
func mergeSame(items []Item, order []string, ident identityOf, people aliases) []Item {
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
		for _, k := range keys(it, ident, people) {
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
		for _, k := range keys(it, ident, people) {
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
				card.Alts = append(card.Alts, source.Alt{
					Source: m.Source, ID: m.ID, URL: m.URL, Installed: m.Installed,
					Obsolete: m.Obsolete, Broken: m.Broken,
					Endorsements: m.Endorsements, Stars: m.Stars, Downloads: m.Downloads,
					Version: m.Version, Updated: m.Updated, Size: m.Size,
				})
			}
		}
		out = append(out, card)
	}
	markReuploads(out, people)
	return out
}

// foldName keeps a name's letters and digits in lower case, so "More Company", "MoreCompany" and "More_Company" are one.
func foldName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// aliases maps a folded author name to the folded first name of the person the catalog says it belongs to.
type aliases map[string]string

func aliasesOf(list []components.AuthorAliases) aliases {
	out := aliases{}
	for _, a := range list {
		if len(a.Names) == 0 {
			continue
		}
		first := foldName(a.Names[0])
		for _, n := range a.Names {
			out[foldName(n)] = first
		}
	}
	return out
}

// authors are a hit's authors (a site may list several, comma-separated), folded and resolved through the aliases.
func (people aliases) authors(author string) []string {
	var out []string
	for part := range strings.SplitSeq(author, ",") {
		a := foldName(part)
		if canon, ok := people[a]; ok {
			a = canon
		}
		if a != "" && !slices.Contains(out, a) {
			out = append(out, a)
		}
	}
	return out
}

// markReuploads marks a card that shares its name with a card from another site by another author and more downloads:
// it may be a copy of that mod uploaded by someone else.
func markReuploads(cards []Item, people aliases) {
	for i := range cards {
		c := &cards[i]
		for _, o := range cards {
			if o.Source == c.Source || o.Downloads <= c.Downloads || foldName(o.Name) == "" ||
				foldName(o.Name) != foldName(c.Name) || sharesAuthor(people.authors(o.Author), people.authors(c.Author)) {
				continue
			}
			c.ReuploadOf = o.Name + " by " + o.Author
			break
		}
	}
}

func sharesAuthor(a, b []string) bool {
	return slices.ContainsFunc(a, func(x string) bool { return slices.Contains(b, x) })
}

func hasSource(members []Item, src string) bool {
	for _, m := range members {
		if m.Source == src {
			return true
		}
	}
	return false
}
