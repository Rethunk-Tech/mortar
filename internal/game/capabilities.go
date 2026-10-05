package game

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// SaveFolder is a game whose saves Mortar can list and back up.
type SaveFolder interface {
	// SavesDir is the saves folder for the selected install store.
	SavesDir(store, home string) (string, error)
}

// StartupSettings is a game that reads startup preferences from a file outside the install, which Mortar
// applies for a launch and restores afterwards.
type StartupSettings interface {
	StartupPreferencesPath(home string) (string, error)
}

// SavesDir returns the saves folder of game id, or a NotFound error when the game has none.
func SavesDir(id, store, home string) (string, error) {
	sf, ok := Find(id).(SaveFolder)
	if !ok {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("game %q has no save folder", id))
	}
	return sf.SavesDir(store, home)
}

// HasSaves reports whether game id has a save folder.
func HasSaves(id string) bool {
	_, ok := Find(id).(SaveFolder)
	return ok
}
