// Package source is the registry of places Mortar finds mods. A driver implements Source and whichever optional
// capabilities (Searcher, Schemer, Hoster) it supports; callers find a capability by type assertion.
package source

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

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

// ErrBusy matches every *BusyError, whichever source refused.
var ErrBusy = errors.New("source is busy")

// BusyError is a source refusing calls for now (403/429). Reset is when it works again, zero when it did not say; the
// queue waits until then instead of failing the item.
type BusyError struct {
	Source string
	Reset  time.Time
}

func (e *BusyError) Error() string { return e.Source + " is busy, try again in a minute" }

func (e *BusyError) Is(target error) bool { return target == ErrBusy }

// Busy is name refusing resp, waiting as long as its Retry-After asks (seconds or an HTTP date) when it says.
func Busy(name string, resp *http.Response) *BusyError {
	e := &BusyError{Source: name}
	v := resp.Header.Get("Retry-After")
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		e.Reset = time.Now().Add(time.Duration(n) * time.Second)
	} else if t, err := http.ParseTime(v); err == nil {
		e.Reset = t
	}
	return e
}

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

// Sort orders a search; "" is best match. A source honours only the sorts its Sorter lists, in its own API, so a
// page is never re-sorted client-side. SortEndorsements is the source's community score (Nexus endorsements,
// Thunderstore rating, Modrinth follows, CurseForge popularity).
const (
	SortDownloads    = "downloads"
	SortUpdated      = "updated"
	SortNewest       = "newest"
	SortEndorsements = "endorsements"
	SortName         = "name"
	SortStars        = "stars"
	SortForks        = "forks"
)

// Sorter is a source that sorts a search server-side; Sorts lists the Sort values it honours besides best match.
type Sorter interface {
	Sorts() []string
}

// SortsOf lists the sorts src honours, none when it has no sorting.
func SortsOf(src Source) []string {
	if s, ok := src.(Sorter); ok {
		return s.Sorts()
	}
	return []string{}
}

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
	// Bundled marks a mod Mortar installs itself with the loader (its companion), so Add does not apply.
	Bundled bool `json:"bundled"`
	// External marks a mod whose author only allows downloads on the site's own page, so Add does not apply.
	External bool `json:"external,omitempty"`
}

// Alt is the same mod on another source: where to open it and install it from, and that hit's own flags.
type Alt struct {
	Source    string `json:"source"`
	ID        string `json:"id"`
	URL       string `json:"url"`
	Installed bool   `json:"installed"`
	Obsolete  bool   `json:"obsolete"`
	Broken    bool   `json:"broken"`
	Loader    bool   `json:"loader"`
	// Endorsements, Stars and Downloads are that site's own counts for the mod.
	Endorsements int `json:"endorsements"`
	Stars        int `json:"stars"`
	Downloads    int `json:"downloads"`
}

// Page is one slice of search hits. A merged search across sources also sets Pages, the page count of its largest
// source, and Failed, the names of sources that did not answer.
type Page struct {
	Total int `json:"total"`
	Pages int `json:"pages,omitempty"`
	// Hidden counts hits on this page the player's browse filters left out; Total still counts them.
	Hidden int    `json:"hidden,omitempty"`
	Items  []Item `json:"items"`
	// Failed are the ids of the sources that did not answer.
	Failed []string `json:"failed,omitempty"`
}

// Searcher is a source that can list mods matching text.
type Searcher interface {
	Search(ctx context.Context, q Query) (Page, error)
}

// ItemLister is a source that holds its whole listing and can answer for packages named by id in one pass. The
// result is keyed by lower-cased id; an id the listing lacks is absent.
type ItemLister interface {
	Items(ctx context.Context, key, mortarVersion string, ids []string) (map[string]Item, error)
}

// VersionRef names one version of a package or project.
type VersionRef struct{ ID, Version string }

// DependencyLister is a source whose cached listing carries the dependencies of its package versions, so a check can
// read them without asking the site per mod. The answer maps each ref the listing knows to the ids of the packages
// that version depends on, ignoring the versions they name; a ref the listing lacks is absent.
type DependencyLister interface {
	Dependencies(ctx context.Context, key, mortarVersion string, refs []VersionRef) (map[VersionRef][]string, error)
}

// InstalledFile is a download installed from a source: its project id, version, and the file's "sha512:<hex>" digest.
type InstalledFile struct{ ID, Version, Digest string }

// Latest is the newest version a source offers in place of an installed file. VersionID names that exact version,
// which a version number alone may not when a project publishes one per loader.
type Latest struct{ ProjectID, Version, VersionID, URL string }

// UpdateChecker is a source that finds the newest version of many installed files in one request, narrowed to the
// game source's loaders and game versions. The answer is keyed by digest; a file already the newest, or one the site
// does not know, is absent.
type UpdateChecker interface {
	Latest(ctx context.Context, src components.GameSource, mortarVersion string, files []InstalledFile) (map[string]Latest, error)
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

// Unavailable is why src cannot be used now, or "" when it can. Drivers return it as an error string (lowercase);
// it reaches the UI as a sentence.
func Unavailable(src Source) string {
	g, ok := src.(Gated)
	if !ok {
		return ""
	}
	reason := g.Unavailable()
	if reason == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(reason)
	return string(unicode.ToUpper(r)) + reason[n:]
}

// Schemer is a source whose links open Mortar through a URL scheme.
type Schemer interface {
	Schemes() []string
}

// LinkOptIn is a schemer that does not claim its scheme from the system unless the user opts in. A source that does
// not implement it is always claimed.
type LinkOptIn interface {
	HandleLinksDefault() bool
}

// PageLinker is a source whose mods each have a page on the site; gameKey is the game's key at the source and id the
// mod's id there.
type PageLinker interface {
	ModPageURL(gameKey, id string) string
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
