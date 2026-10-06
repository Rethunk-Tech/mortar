// Package thunderstore is the Thunderstore source driver: search over a community's chunked package index, ror2mm
// links, and package resolution for installs.
package thunderstore

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// BaseURL is the real site.
const BaseURL = "https://thunderstore.io"

// Driver searches Thunderstore. The zero value talks to the real site and caches under the user cache directory.
type Driver struct {
	HTTP *http.Client
	URL  string
	// CacheDir is Mortar's cache directory; the index lives in its thunderstore folder.
	CacheDir string
	// Now is the clock the hourly refresh reads; nil means time.Now.
	Now func() time.Time
}

var _ = source.Register(Driver{})

// ID is the catalog id.
func (Driver) ID() string { return "thunderstore" }

// Name is the site's name.
func (Driver) Name() string { return "Thunderstore" }

// Modes lists both ways: Mortar downloads packages itself, and ror2mm links hand one over.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download, source.Handoff} }

// ModPageURL is the package's page; id is Namespace-Name, and a namespace holds no dash.
func (d Driver) ModPageURL(gameKey, id string) string {
	owner, name, ok := strings.Cut(id, "-")
	if !ok {
		return ""
	}
	base := d.URL
	if base == "" {
		base = BaseURL
	}
	return base + "/c/" + gameKey + "/p/" + owner + "/" + name + "/"
}

// score ranks a package against the query tokens: name over owner over summary; zero when a token matches nowhere.
func score(p pkg, tokens []string) int {
	name, owner, summary := strings.ToLower(p.Name), strings.ToLower(p.Owner), strings.ToLower(p.Summary)
	total := 0
	for _, t := range tokens {
		switch {
		case name == t:
			total += 100
		case strings.HasPrefix(name, t):
			total += 60
		case strings.Contains(name, t):
			total += 40
		case strings.Contains(owner, t):
			total += 20
		case strings.Contains(summary, t):
			total += 5
		default:
			return 0
		}
	}
	return total
}

// Search lists the community's packages matching the text, best match first and most downloaded among equals.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	if q.Key == "" {
		return source.Page{}, fmt.Errorf("game %q has no Thunderstore community", q.Game)
	}
	pk, err := d.packages(ctx, q.Key, source.UserAgent(q.Version)+" (+https://mortar.rethunk.tech)")
	if err != nil {
		return source.Page{}, err
	}
	tokens := strings.Fields(strings.ToLower(q.Text))
	type hit struct {
		p     pkg
		score int
	}
	var hits []hit
	for _, p := range pk {
		if !source.CategoryMatch(p.Categories, q.Categories, q.ExcludeCategories) {
			continue
		}
		if s := score(p, tokens); s > 0 || len(tokens) == 0 {
			hits = append(hits, hit{p, s})
		}
	}
	slices.SortStableFunc(hits, func(a, b hit) int {
		switch q.Sort {
		case source.SortDownloads:
			return b.p.Downloads - a.p.Downloads
		case source.SortEndorsements:
			return b.p.Rating - a.p.Rating
		case source.SortUpdated:
			return strings.Compare(b.p.Updated, a.p.Updated)
		case source.SortName:
			return strings.Compare(strings.ToLower(a.p.Name), strings.ToLower(b.p.Name))
		}
		if a.score != b.score {
			return b.score - a.score
		}
		return b.p.Downloads - a.p.Downloads
	})
	start := max(q.Page-source.FirstPage, 0) * source.PageSize
	end := min(start+source.PageSize, len(hits))
	items := make([]source.Item, 0, max(end-start, 0))
	for i := start; i < end; i++ {
		p := hits[i].p
		items = append(items, source.Item{
			Source: d.ID(), ID: p.Owner + "-" + p.Name, Name: p.Name, Summary: p.Summary, Author: p.Owner,
			Version: p.Versions[0].Number, Picture: p.Icon, Endorsements: p.Rating, Downloads: p.Downloads,
			Updated: p.Updated, URL: p.URL, Adult: p.Adult, Repo: p.Repo, Obsolete: p.Hidden,
		})
	}
	return source.Page{Total: len(hits), Items: items}, nil
}

// Categories lists the community's package categories, sorted.
func (d Driver) Categories(ctx context.Context, key string) ([]string, error) {
	pk, err := d.packages(ctx, key, source.UserAgent("")+" (+https://mortar.rethunk.tech)")
	if err != nil {
		return nil, err
	}
	var all []string
	for _, p := range pk {
		all = append(all, p.Categories...)
	}
	return source.UniqueNames(all), nil
}

// Deprecation is the index's word on a package its author deprecated. Replacement is another listed package the
// deprecated package's own summary points to, as "Namespace-Name" or as a bare name of the same author's, or empty:
// Thunderstore has no replacement field.
type Deprecation struct {
	Replacement string
}

var replaceHint = regexp.MustCompile(`(?i)\b(deprecated|replaced|superseded|obsolete|moved|use)\b`)

// Deprecated lists the community's deprecated packages by lower-cased "Namespace-Name", from the cached index.
func (d Driver) Deprecated(ctx context.Context, key, version string) (map[string]Deprecation, error) {
	pk, err := d.packages(ctx, key, source.UserAgent(version)+" (+https://mortar.rethunk.tech)")
	if err != nil {
		return nil, err
	}
	live := map[string]string{}
	for _, p := range pk {
		if !p.Hidden {
			live[strings.ToLower(p.Owner+"-"+p.Name)] = p.Owner + "-" + p.Name
		}
	}
	out := map[string]Deprecation{}
	for _, p := range pk {
		if !p.Hidden {
			continue
		}
		self := strings.ToLower(p.Owner + "-" + p.Name)
		dep := Deprecation{}
		if replaceHint.MatchString(p.Summary) {
			toks := strings.FieldsFunc(p.Summary, func(r rune) bool { return strings.ContainsRune(" \t\n,;:()[]\"'", r) })
			// Authors mostly name the successor bare ("use WeatherInjector instead"); a bare name is only trusted
			// among the author's own packages, since across the whole index it matches common words.
			for _, prefix := range []string{"", strings.ToLower(p.Owner) + "-"} {
				for _, tok := range toks {
					if name, ok := live[prefix+strings.ToLower(strings.Trim(tok, ".!?"))]; ok && strings.ToLower(name) != self {
						dep.Replacement = name
						break
					}
				}
				if dep.Replacement != "" {
					break
				}
			}
		}
		out[self] = dep
	}
	return out, nil
}
