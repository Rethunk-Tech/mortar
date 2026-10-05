// Package source is the registry of places Mortar finds mods. A driver implements Source and whichever optional
// capabilities (Searcher, Schemer, PageLinker, Hoster) it supports; callers find a capability by type assertion.
package source

import (
	"context"
	"errors"
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
}

// Page is one slice of search hits.
type Page struct {
	Total int    `json:"total"`
	Items []Item `json:"items"`
}

// Searcher is a source that can list mods matching text.
type Searcher interface {
	Search(ctx context.Context, q Query) (Page, error)
}

// Schemer is a source whose links open Mortar through a URL scheme.
type Schemer interface {
	Schemes() []string
}

// PageLinker is a source that can name a mod's web page.
type PageLinker interface {
	ModPageURL(gameKey string, modID int) string
}

// Hoster is a source that owns web hosts, so a bare URL can be traced back to it.
type Hoster interface {
	Hosts() []string
}

var (
	mu       sync.RWMutex
	registry = map[string]Source{}
)

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

// Schemes lists every URL scheme a registered source claims.
func Schemes() []string {
	var out []string
	for _, s := range All() {
		if sc, ok := s.(Schemer); ok {
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
