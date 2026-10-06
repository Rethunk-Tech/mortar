package components

import "sync/atomic"

// active is the client whose verified manifest the whole process reads the game catalog from; every service that
// loads one selects it, possibly at the same time.
var active atomic.Pointer[Client]

// Use selects the client whose manifest answers Active, Games and Game. Until one is selected, and while its
// manifest lists no games, the manifest compiled into Mortar answers.
func Use(c *Client) { active.Store(c) }

// Active is the selected client, or nil.
func Active() *Client { return active.Load() }

// Games is the game catalog in use: the selected manifest's, else the bundled one's.
func Games() []GameInfo {
	if c := active.Load(); c != nil {
		if g := c.Manifest().Games; len(g) > 0 {
			return g
		}
	}
	m, err := BundledManifest()
	if err != nil {
		return nil
	}
	return m.Games
}

// Game is the catalog entry with id, from the selected manifest when it lists the game, else the bundled one.
func Game(id string) (GameInfo, bool) {
	if c := active.Load(); c != nil {
		return c.Game(id)
	}
	return BundledGame(id)
}
