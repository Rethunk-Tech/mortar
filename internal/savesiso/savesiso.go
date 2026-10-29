// Package savesiso gives a profile its own save folder for the length of a launch. Apply sets the game's shared save
// folder aside and points that path at the profile's folder (a symlink, or a working copy where links are refused);
// Purge undoes it. The record is persisted before the shared folder moves, so a crash is finished by Recover.
package savesiso

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// heldSuffix names the shared folder while it is set aside, beside the game's save path so the move is a rename.
const heldSuffix = ".mortar-shared"

const journalFile = "saves.json"

// ErrUnrecovered refuses a new swap over the record of one that was never undone.
var ErrUnrecovered = errors.New("an earlier save swap has not been undone")

// Manifest is the persisted record of one swap.
type Manifest struct {
	Journal string `json:"journal"`
	// Saves is the game's save path and Profile the profile's own folder.
	Saves   string `json:"saves"`
	Profile string `json:"profile"`
	// Held is where the shared folder waits, "" when the game had none.
	Held string `json:"held,omitempty"`
	// Copied means Saves is a working copy of Profile instead of a link; Live is set once that copy is complete,
	// and only a complete copy is written back.
	Copied bool `json:"copied,omitempty"`
	Live   bool `json:"live,omitempty"`
	// WrittenBack is set once Profile holds the working copy, so a Purge run again after the copy was partly removed
	// does not write that remainder over it.
	WrittenBack bool `json:"writtenBack,omitempty"`
}

func journalPath(dir string) string { return filepath.Join(dir, journalFile) }

// HasJournal reports whether dir holds the record of a save swap not yet undone.
func HasJournal(dir string) bool {
	_, err := os.Stat(journalPath(dir))
	return err == nil
}

func persist(m Manifest) error {
	return datadir.WriteJSONIn(m.Journal, journalFile, m)
}

func isLink(p string) bool {
	info, err := os.Lstat(p)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// Apply swaps profile's folder in for saves. noLink skips the symlink and uses a copy.
func Apply(journal, saves, profile string, noLink bool) (Manifest, error) {
	if exists(journalPath(journal)) {
		return Manifest{}, fmt.Errorf("%w: recover %s first", ErrUnrecovered, journal)
	}
	if err := os.MkdirAll(profile, 0o700); err != nil {
		return Manifest{}, err
	}
	m := Manifest{Journal: journal, Saves: saves, Profile: profile}
	if exists(saves) {
		m.Held = saves + heldSuffix
		if exists(m.Held) {
			return Manifest{}, fmt.Errorf("%s is already set aside", m.Held)
		}
	}
	if err := persist(m); err != nil {
		return Manifest{}, err
	}
	if m.Held != "" {
		if err := safefs.Rename(saves, m.Held); err != nil {
			_ = safefs.RemoveAll(journal)
			return Manifest{}, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(saves), 0o750); err != nil {
		return m, undoOnError(m, err)
	}
	if !noLink && os.Symlink(profile, saves) == nil {
		return m, nil
	}
	m.Copied = true
	if err := persist(m); err != nil {
		return m, undoOnError(m, err)
	}
	if err := os.MkdirAll(saves, 0o750); err != nil {
		return m, undoOnError(m, err)
	}
	if err := datadir.CopyTree(profile, saves); err != nil {
		return m, undoOnError(m, err)
	}
	m.Live = true
	if err := persist(m); err != nil {
		return m, undoOnError(m, err)
	}
	return m, nil
}

func undoOnError(m Manifest, cause error) error {
	return errors.Join(cause, Purge(m))
}

// Purge undoes the swap: a working copy is written back to the profile, the link or copy is removed, and the shared
// folder returns. Every step checks what is on disk, so it can run again after a crash.
func Purge(m Manifest) error {
	// The record on disk is the authority: with it gone the swap was already undone, and replaying a stale manifest
	// would remove the profile's saves or move the held folder over what the game has written since.
	if !HasJournal(m.Journal) {
		return nil
	}
	switch {
	case m.Copied && m.Live && exists(m.Saves) && !isLink(m.Saves):
		if !m.WrittenBack {
			next := m.Profile + ".new"
			_ = safefs.RemoveAll(next)
			if err := datadir.CopyTree(m.Saves, next); err != nil {
				return err
			}
			if err := safefs.RemoveAll(m.Profile); err != nil {
				return err
			}
			if err := safefs.Rename(next, m.Profile); err != nil {
				return err
			}
			m.WrittenBack = true
			if err := persist(m); err != nil {
				return err
			}
		}
		if err := safefs.RemoveAll(m.Saves); err != nil {
			return err
		}
	case m.Copied && exists(m.Saves) && !isLink(m.Saves):
		if err := safefs.RemoveAll(m.Saves); err != nil {
			return err
		}
	case isLink(m.Saves):
		if err := safefs.Remove(m.Saves); err != nil {
			return err
		}
	}
	if m.Held != "" && exists(m.Held) {
		if exists(m.Saves) {
			return fmt.Errorf("%s is in the way of the shared saves kept at %s", m.Saves, m.Held)
		}
		if err := safefs.Rename(m.Held, m.Saves); err != nil {
			return err
		}
	}
	return safefs.RemoveAll(m.Journal)
}

// Recover finishes an unfinished swap found at journal: nothing while alive reports the game running, else Purge.
func Recover(journal string, alive func() bool) error {
	var m Manifest
	if found, err := datadir.ReadJournal(journalPath(journal), alive, &m); !found || err != nil {
		return err
	}
	m.Journal = journal
	return Purge(m)
}
