package thunderstore

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	tstore "github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Look is how the Mods tab shows a package: its icon's address, the one of its categories it is grouped by, and every
// category the site gives it, in the site's order.
type Look struct {
	Icon       string
	Category   string
	Categories []string
}

// lookOf is the Look of a package read from the community's listing.
func lookOf(p *tstore.Package) Look {
	return Look{Icon: p.Icon, Category: Category(p.Categories), Categories: slices.Clone(p.Categories)}
}

// CategoryNames is the community's category name of each slug a project declares for its upload ("clientside" is
// "Client-side"), read from the stored listing without the network. A slug the listing has no name for reads as its
// own words, capitalised.
func (d Driver) CategoryNames(key string, slugs []string) []string {
	var known []string
	if communityKey.MatchString(key) {
		known = readDerived(filepath.Join(d.cacheRoot(), "thunderstore"), key).Categories
	}
	out := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		i := slices.IndexFunc(known, func(n string) bool { return categoryFold(n) == categoryFold(slug) })
		if i >= 0 {
			out = append(out, known[i])
			continue
		}
		words := strings.Fields(strings.ReplaceAll(slug, "-", " "))
		for j, w := range words {
			words[j] = strings.ToUpper(w[:1]) + w[1:]
		}
		out = append(out, strings.Join(words, " "))
	}
	return out
}

// categoryFold is a category name or slug with case, punctuation and spacing dropped, so the name and the slug the
// site derives from it compare equal.
func categoryFold(s string) string {
	s = strings.ReplaceAll(strings.ToLower(s), "&", "and")
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// broadCategories fit nearly every package of a community, or say where it runs rather than what it is.
var broadCategories = []string{"mods", "bepinex", "melonloader", "client-side", "server-side", "ai generated"}

// Category is the one of a package's site categories that says what it is: the first that is not a broad one, else
// the first.
func Category(cats []string) string {
	for _, c := range cats {
		if !slices.Contains(broadCategories, strings.ToLower(c)) {
			return c
		}
	}
	if len(cats) > 0 {
		return cats[0]
	}
	return ""
}

var (
	errNotStored = errors.New("no cached Thunderstore listing")
	// communityKey keeps a key that is a path out of the file names Mortar keeps beside the listing.
	communityKey = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

// CachedIcons returns the icon address of each package named (Namespace-Name, any case) that the community's cached
// listing holds, keyed by the lower-case name. Unlike CachedLooks it keeps no record of what it was asked, so asking
// again answers again. It never asks the network.
func (d Driver) CachedIcons(key string, names []string) (map[string]string, error) {
	c, err := d.client("")
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, n := range names {
		want[strings.ToLower(n)] = true
	}
	out := map[string]string{}
	stored, err := c.Stored(context.Background(), key, func(p *tstore.Package) {
		if id := p.ID(); want[id] && p.Icon != "" {
			out[id] = p.Icon
		}
	})
	if err != nil || !stored {
		return nil, cmp.Or(err, errNotStored)
	}
	return out, nil
}

// lookedUp is the names CachedLooks already looked up in the stored listing. A new listing removes the file (Built).
// Format is lookedFormat: a record without it was stamped with an index hash this code cannot check against the
// stored listing, so it is read as absent.
type lookedUp struct {
	Format int      `json:"format"`
	Names  []string `json:"names"`
}

const lookedFormat = 2

func lookedPath(dir, key string) string { return filepath.Join(dir, key+".looked.json") }

// CachedLooks returns the Look of each package named (Namespace-Name, any case) that the community's cached listing
// holds, keyed by the lower-case name. It never asks the network, however old the listing is. Names already looked up
// in this listing are not looked up again: a package added from disk is never in it, and decoding the listing costs
// most of a CPU second.
func (d Driver) CachedLooks(key string, names []string) (map[string]Look, error) {
	if !communityKey.MatchString(key) {
		return nil, errNotStored
	}
	c, err := d.client("")
	if err != nil {
		return nil, err
	}
	markPath := lookedPath(filepath.Join(d.cacheRoot(), "thunderstore"), key)
	var mark lookedUp
	if b, err := safefs.ReadFile(markPath); err == nil && (json.Unmarshal(b, &mark) != nil || mark.Format != lookedFormat) {
		mark = lookedUp{}
	}
	mark.Format = lookedFormat
	want := map[string]bool{}
	for _, n := range names {
		if id := strings.ToLower(n); !slices.Contains(mark.Names, id) {
			want[id] = true
		}
	}
	if len(want) == 0 {
		return map[string]Look{}, nil
	}
	out := map[string]Look{}
	stored, err := c.Stored(context.Background(), key, func(p *tstore.Package) {
		if id := p.ID(); want[id] {
			out[id] = lookOf(p)
		}
	})
	if err != nil || !stored {
		return nil, cmp.Or(err, errNotStored)
	}
	for id := range want {
		mark.Names = append(mark.Names, id)
	}
	slices.Sort(mark.Names)
	_ = datadir.WriteJSONCompact(markPath, mark)
	return out, nil
}
