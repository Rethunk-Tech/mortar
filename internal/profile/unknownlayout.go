package profile

import (
	"fmt"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// What an install stopped for: the game's installer found no layout for the archive, or it noted something Vortex asks
// the player to accept (an old core mod, files that overwrite config).
const (
	ReasonUnrecognised   = "unrecognised"
	ReasonDeprecatedCore = "deprecatedCore"
	ReasonProtectedPaths = "protectedPaths"
)

// UnknownLayoutError is an install stopped for the player to accept: Reason says what, and Files are the ones it is
// about (the whole archive for a layout no installer took), for the player to judge by.
type UnknownLayoutError struct {
	Game   string   `json:"game"`
	Key    string   `json:"key"`
	Reason string   `json:"reason"`
	Files  []string `json:"files"`
	// Overridable is set when the player may install it anyway; unknownLayout refuse allows no override for an
	// archive with no layout.
	Overridable bool `json:"overridable"`
}

// Detail is what the GUI reads off the error (usererr.Detailer).
func (e *UnknownLayoutError) Detail() any { return e }

func (e *UnknownLayoutError) Error() string {
	switch {
	case e.Reason == ReasonDeprecatedCore:
		return "This is an old version of a core mod; installing it is your call"
	case e.Reason == ReasonProtectedPaths:
		return fmt.Sprintf("This mod writes to %d files that other mods or you may have changed; installing it is your call", len(e.Files))
	case e.Overridable:
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

// checkUnknownLayout stops an install that Vortex would ask the player about, as the game's unknownLayout setting says:
// install never stops one, ask stops each, and refuse stops an archive with no layout for good and asks about the rest.
// A store without the setting installs everything.
func (s *Store) checkUnknownLayout(game, id, key string, l installer.Layout) error {
	reason, files := installQuestion(l)
	if reason == "" || s.UnknownLayoutMode == nil {
		return nil
	}
	mode := s.UnknownLayoutMode(game)
	if mode == "install" || s.layoutOK.take(game, id, key) {
		return nil
	}
	return &UnknownLayoutError{Game: game, Key: key, Reason: reason, Files: files, Overridable: mode == "ask" || reason != ReasonUnrecognised}
}

// installQuestion is what the player is asked to accept about l, with its files, or "" when nothing.
func installQuestion(l installer.Layout) (reason string, files []string) {
	if l.Unrecognised {
		return ReasonUnrecognised, noteFiles(l, installer.NoteUnrecognised, true)
	}
	for _, r := range []struct{ reason, code string }{
		{ReasonDeprecatedCore, installer.NoteDeprecatedCore}, {ReasonProtectedPaths, installer.NoteProtectedPaths},
	} {
		if files := noteFiles(l, r.code, false); files != nil {
			return r.reason, files
		}
	}
	return "", nil
}

// noteFiles are the files the notes of code list, or every file the layout places when none are and all is set; nil
// when l has no note of code.
func noteFiles(l installer.Layout, code string, all bool) []string {
	var out []string
	var found bool
	for _, n := range l.Notes {
		if n.Code == code {
			found = true
			out = append(out, n.Files...)
		}
	}
	if (found || all) && len(out) == 0 {
		for _, f := range l.Files {
			out = append(out, f.Src)
		}
	}
	if !found && !all {
		return nil
	}
	return out
}
