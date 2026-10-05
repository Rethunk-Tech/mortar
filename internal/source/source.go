// Package source is the registry of places Mortar finds mods. A driver implements Source and whichever optional
// capabilities (Searcher, Schemer, PageLinker, Hoster) it supports; callers find a capability by type assertion.
package source

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// Acquire is how a source's mods reach a profile.
type Acquire string

const (
	// Download means Mortar fetches the file itself.
	Download Acquire = "download"
	// Handoff means the user finishes in a browser or another app.
	Handoff Acquire = "handoff"
)

// ErrBusy means GitHub refused the search (403/429); the caller should wait a minute.
var ErrBusy = errors.New("GitHub is busy, try again in a minute")

// Source is one mod site.
type Source interface {
	ID() string
	Name() string
	Modes() []Acquire
}

// Query is one search.
type Query struct {
	// Game is the catalog game id and Key the source's name for it.
	Game    string
	Key     string
	Text    string
	Page    int
	Version string
	// Categories keeps hits in any of these categories; ExcludeCategories drops hits in any of those. Names match
	// case-insensitively. Sources without categories ignore them.
	Categories        []string
	ExcludeCategories []string
	// Sort is one of the Sort constants; empty keeps the source's own order.
	Sort string
}

// Sort orders a search; each source maps it to what it can.
const (
	SortDownloads    = "downloads"
	SortUpdated      = "updated"
	SortEndorsements = "endorsements"
	SortName         = "name"
)

// Item is one search hit.
type Item struct {
	Source       string `json:"source"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Summary      string `json:"summary"`
	Author       string `json:"author"`
	Version      string `json:"version"`
	Picture      string `json:"picture"`
	Endorsements int    `json:"endorsements"`
	Stars        int    `json:"stars"`
	Downloads    int    `json:"downloads"`
	Updated      string `json:"updated"`
	URL          string `json:"url"`
	Installed    bool   `json:"installed"`
	// Repo is the mod's GitHub repository as "owner/repo" when its site links one; it ties the same mod on
	// several sources together.
	Repo string `json:"repo,omitempty"`
	// Alts are the same mod's hits on the game's other sources, set by a merged search that found it on several.
	Alts []Alt `json:"alts,omitempty"`
	// Adult marks a mod its site flags as adult content; browse hides it unless the player opted in.
	Adult bool `json:"adult"`
	// Obsolete marks a mod its author or site calls obsolete or deprecated, and Broken one the game's compatibility
	// list calls broken; Loader marks the game's mod loader itself, which Mortar installs apart from profiles.
	Obsolete bool `json:"obsolete"`
	Broken   bool `json:"broken"`
	Loader   bool `json:"loader"`
}

// Alt is the same mod on another source: where to open it and install it from.
type Alt struct {
	Source    string `json:"source"`
	ID        string `json:"id"`
	URL       string `json:"url"`
	Installed bool   `json:"installed"`
}

// Page is one slice of search hits. A merged search across sources also sets Pages, the page count of its largest
// source, and Failed, the names of sources that did not answer.
type Page struct {
	Total int `json:"total"`
	Pages int `json:"pages,omitempty"`
	// Hidden counts hits on this page the player's browse filters left out; Total still counts them.
	Hidden int      `json:"hidden,omitempty"`
	Items  []Item   `json:"items"`
	Failed []string `json:"failed,omitempty"`
}

// Searcher is a source that can list mods matching text.
type Searcher interface {
	Search(ctx context.Context, q Query) (Page, error)
}

// Categorizer is a source whose mods carry categories that Query.Categories filters on.
type Categorizer interface {
	Categories(ctx context.Context, key string) ([]string, error)
}

// Gated is a source that cannot be used until the user sets something up; Unavailable says what, or is empty when
// the source is ready.
type Gated interface {
	Unavailable() string
}

// Unavailable is why src cannot be used now, or "" when it can.
func Unavailable(src Source) string {
	if g, ok := src.(Gated); ok {
		return g.Unavailable()
	}
	return ""
}

// Schemer is a source whose links open Mortar through a URL scheme.
type Schemer interface {
	Schemes() []string
}

// PageLinker is a source that can name a mod's web page from its native id (Item.ID).
type PageLinker interface {
	ModPageURL(gameKey, id string) string
}

// LinkOptIn is a schemer that does not claim its scheme from the system unless the user opts in. A source that does
// not implement it is always claimed.
type LinkOptIn interface {
	HandleLinksDefault() bool
}

// Hoster is a source that owns web hosts, so a bare URL can be traced back to it.
type Hoster interface {
	Hosts() []string
}

var (
	mu       sync.RWMutex
	registry = map[string]Source{}
	// handleLinks is the user's per-source choice to claim the source's link scheme, over its default.
	handleLinks = map[string]bool{}
)

// SetHandleLinks records which sources the user chose to claim link schemes for (true) or not (false); a source
// absent from the map keeps its default.
func SetHandleLinks(choice map[string]bool) {
	mu.Lock()
	defer mu.Unlock()
	handleLinks = maps.Clone(choice)
}

// SetHandleLink records one source's choice to claim its link scheme.
func SetHandleLink(id string, on bool) {
	mu.Lock()
	defer mu.Unlock()
	if handleLinks == nil {
		handleLinks = map[string]bool{}
	}
	handleLinks[id] = on
}

// SchemesOf lists the link schemes the source id has, whether or not it claims them now.
func SchemesOf(id string) []string {
	e, ok := Get(id)
	if !ok {
		return nil
	}
	if sc, ok := e.Source.(Schemer); ok {
		return sc.Schemes()
	}
	return nil
}

// OptsIn reports whether the source claims its scheme only when the user chooses it.
func OptsIn(id string) bool {
	e, ok := Get(id)
	if !ok {
		return false
	}
	opt, ok := e.Source.(LinkOptIn)
	return ok && !opt.HandleLinksDefault()
}

// Claims reports whether the source's link scheme is claimed from the system now: the user's choice, else its default.
func Claims(id string) bool {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := registry[strings.ToLower(strings.TrimSpace(id))]
	return ok && claims(s)
}

func claims(s Source) bool {
	if on, ok := handleLinks[s.ID()]; ok {
		return on
	}
	opt, ok := s.(LinkOptIn)
	return !ok || opt.HandleLinksDefault()
}

// Register adds a driver and reports true; a driver package calls it in a blank package variable, so importing the
// package registers it.
func Register(s Source) bool {
	mu.Lock()
	defer mu.Unlock()
	registry[s.ID()] = s
	return true
}

// Entry is a registered source. Ask Source for a capability by type assertion.
type Entry struct{ Source Source }

// Get returns the registered source with this id.
func Get(id string) (Entry, bool) {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := registry[strings.ToLower(strings.TrimSpace(id))]
	return Entry{Source: s}, ok
}

// All lists every registered source ordered by id.
func All() []Source {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Source, 0, len(registry))
	for _, s := range registry {
		out = append(out, s)
	}
	slices.SortFunc(out, func(a, b Source) int { return strings.Compare(a.ID(), b.ID()) })
	return out
}

// ForGame lists the game's sources in catalog order; a catalog entry without a registered driver is left out.
func ForGame(g components.GameInfo) []Source {
	var out []Source
	for _, gs := range g.Sources {
		if e, ok := Get(gs.ID); ok {
			out = append(out, e.Source)
		}
	}
	return out
}

// Searchable lists the game's sources that can search, in catalog order.
func Searchable(g components.GameInfo) []Source {
	return slices.DeleteFunc(ForGame(g), func(s Source) bool {
		_, ok := s.(Searcher)
		return !ok
	})
}

// Schemes lists every URL scheme a registered source claims from the system.
func Schemes() []string {
	var out []string
	for _, s := range All() {
		mu.RLock()
		on := claims(s)
		mu.RUnlock()
		if sc, ok := s.(Schemer); ok && on {
			out = append(out, sc.Schemes()...)
		}
	}
	return out
}

// IsLink reports whether arg is a link in a scheme some source claims.
func IsLink(arg string) bool {
	lower := strings.ToLower(arg)
	return slices.ContainsFunc(Schemes(), func(scheme string) bool { return strings.HasPrefix(lower, scheme+"://") })
}

// NameOfHost is the name of the source that owns the web host, with a leading "www." ignored.
func NameOfHost(host string) (string, bool) {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	for _, s := range All() {
		if h, ok := s.(Hoster); ok && slices.Contains(h.Hosts(), host) {
			return s.Name(), true
		}
	}
	return "", false
}

// CategoryMatch reports whether a mod with the given categories passes the include and exclude lists.
func CategoryMatch(have, include, exclude []string) bool {
	has := func(want string) bool {
		return slices.ContainsFunc(have, func(h string) bool { return strings.EqualFold(h, want) })
	}
	if slices.ContainsFunc(exclude, has) {
		return false
	}
	return len(include) == 0 || slices.ContainsFunc(include, has)
}

// UniqueNames returns names without empties or case-insensitive repeats, sorted case-insensitively.
func UniqueNames(names []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if k := strings.ToLower(n); n != "" && !seen[k] {
			seen[k] = true
			out = append(out, n)
		}
	}
	slices.SortFunc(out, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return out
}
