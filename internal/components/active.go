package components

import (
	"slices"
	"strings"
	"sync/atomic"
)

// active is the client whose verified manifest the whole process reads the game catalog from; every service that
// loads one selects it, possibly at the same time.
var active atomic.Pointer[Client]

// Use selects the client whose manifest answers Active, Games, Game and GameByNexusDomain. Until one is selected, and while its
// manifest lists no games, the manifest compiled into Mortar answers.
func Use(c *Client) { active.Store(c) }

// Active is the selected client, or nil.
func Active() *Client { return active.Load() }

// Games is the game catalog in use: the selected manifest's, else the bundled one's. A game the selected manifest left
// out for a newer Mortar keeps its bundled entry, so it does not vanish (NeedsNewerMortar).
func Games() []GameInfo {
	if c := active.Load(); c != nil {
		if m := c.Manifest(); len(m.Games) > 0 {
			g := m.Games
			for _, id := range m.NeedsNewer {
				if b, ok := bundledGame(id); ok && !slices.ContainsFunc(g, func(x GameInfo) bool { return x.ID == id }) {
					g = append(slices.Clone(g), b)
				}
			}
			return withUserGames(g)
		}
	}
	m, err := BundledManifest()
	if err != nil {
		return withUserGames(nil)
	}
	return withUserGames(m.Games)
}

// NeedsNewerMortar reports whether the selected manifest has an entry for the game that only a newer Mortar can read.
func NeedsNewerMortar(id string) bool {
	c := active.Load()
	return c != nil && slices.Contains(c.Manifest().NeedsNewer, id)
}

// Authors are the author aliases in use: the selected manifest's, else the bundled one's.
func Authors() []AuthorAliases {
	if c := active.Load(); c != nil {
		if a := c.Manifest().Authors; len(a) > 0 {
			return a
		}
	}
	m, err := BundledManifest()
	if err != nil {
		return nil
	}
	return m.Authors
}

// Game is the catalog entry with id, from the selected manifest when it lists the game, else the bundled one.
func Game(id string) (GameInfo, bool) {
	var g GameInfo
	var ok bool
	if c := active.Load(); c != nil {
		g, ok = c.Game(id)
	} else {
		g, ok = bundledGame(id)
	}
	if !ok {
		return userGame(id)
	}
	return g, true
}

// HasComponent reports whether the catalog in use (else the bundled one) has the game's component of that name, which
// is what Mortar installs for a loader's companion.
func HasComponent(game, name string) bool {
	if c := active.Load(); c != nil {
		_, ok := c.Component(game, name)
		return ok
	}
	m, err := BundledManifest()
	if err != nil {
		return false
	}
	_, ok := findComponent(m.Components, game, name)
	return ok
}

// GameByNexusDomain is the catalog game whose Nexus source key (the v1 URL segment) is domain; the browser extension
// names games by that domain.
func GameByNexusDomain(domain string) (GameInfo, bool) {
	for _, g := range Games() {
		if key := g.NexusDomain(); key != "" && strings.EqualFold(key, domain) {
			return g, true
		}
	}
	return GameInfo{}, false
}
