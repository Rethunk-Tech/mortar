package profile

import (
	"fmt"
	"slices"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// UnknownLayoutError is an install stopped because the game's installer found no layout for the archive and the game's
// unknownLayout setting does not place it as packed. Files are what the archive holds, for the player to judge by.
type UnknownLayoutError struct {
	Game  string   `json:"game"`
	Key   string   `json:"key"`
	Files []string `json:"files"`
	// Overridable is set when the player may install it as packed anyway (unknownLayout ask); refuse allows no override.
	Overridable bool `json:"overridable"`
}

// Detail is what the GUI reads off the error (usererr.Detailer).
func (e *UnknownLayoutError) Detail() any { return e }

func (e *UnknownLayoutError) Error() string {
	if e.Overridable {
		return fmt.Sprintf("Mortar has no layout for this mod (%d files), so it would place every file as packed", len(e.Files))
	}
	return fmt.Sprintf("Mortar has no layout for this mod (%d files), and Settings is set to refuse those", len(e.Files))
}

// layoutAllowed holds the installs the player chose to place as packed, until the install that uses the choice reaches
// its layout.
type layoutAllowed struct {
	mu  sync.Mutex
	set map[string]bool
}

func layoutAllowedKey(game, id, key string) string { return game + "\x00" + id + "\x00" + key }

func (a *layoutAllowed) allow(game, id, key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.set == nil {
		a.set = map[string]bool{}
	}
	a.set[layoutAllowedKey(game, id, key)] = true
}

func (a *layoutAllowed) take(game, id, key string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	k := layoutAllowedKey(game, id, key)
	had := a.set[k]
	delete(a.set, k)
	return had
}

// AllowUnknownLayout is the player's "Install anyway" for an archive stopped by UnknownLayoutError: the next install of
// key into the profile places it as packed.
func (s *Service) AllowUnknownLayout(game, id, key string) { s.store.AllowUnknownLayout(game, id, key) }

// AllowUnknownLayout is Service.AllowUnknownLayout.
func (s *Store) AllowUnknownLayout(game, id, key string) { s.layoutOK.allow(game, id, key) }

// checkUnknownLayout stops an install of a layout no installer shape took, as the game's unknownLayout setting says.
// A store without the setting places it as packed.
func (s *Store) checkUnknownLayout(game, id, key string, l installer.Layout) error {
	if !l.Unrecognised || s.UnknownLayoutMode == nil {
		return nil
	}
	mode := s.UnknownLayoutMode(game)
	if mode == "install" || s.layoutOK.take(game, id, key) {
		return nil
	}
	files := installerNoteFiles(l)
	return &UnknownLayoutError{Game: game, Key: key, Files: files, Overridable: mode == "ask"}
}

// installerNoteFiles are the archive's files the unrecognised-layout note lists, else every file the layout places.
func installerNoteFiles(l installer.Layout) []string {
	for _, n := range l.Notes {
		if n.Code == installer.NoteUnrecognised && len(n.Files) > 0 {
			return slices.Clone(n.Files)
		}
	}
	out := make([]string, len(l.Files))
	for i, f := range l.Files {
		out[i] = f.Src
	}
	return out
}
