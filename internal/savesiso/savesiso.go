// Package savesiso gives a profile its own save folder for the length of a launch. Apply sets the game's shared save
// folder aside and points that path at the profile's folder (a symlink, or a working copy where links are refused);
// Purge undoes it. The record is persisted before the shared folder moves, so a crash is finished by Recover.
package savesiso

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
}

func journalPath(dir string) string { return filepath.Join(dir, journalFile) }

func persist(m Manifest) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.Journal, 0o700); err != nil {
		return err
	}
	tmp := journalPath(m.Journal) + ".tmp"
	if err := fsx.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return fsx.Rename(tmp, journalPath(m.Journal))
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
		if err := os.Rename(saves, m.Held); err != nil {
			_ = os.RemoveAll(journal)
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
	switch {
	case m.Copied && m.Live && exists(m.Saves) && !isLink(m.Saves):
		next := m.Profile + ".new"
		_ = os.RemoveAll(next)
		if err := datadir.CopyTree(m.Saves, next); err != nil {
			return err
		}
		if err := os.RemoveAll(m.Profile); err != nil {
			return err
		}
		if err := os.Rename(next, m.Profile); err != nil {
			return err
		}
		if err := os.RemoveAll(m.Saves); err != nil {
			return err
		}
	case m.Copied && exists(m.Saves) && !isLink(m.Saves):
		if err := os.RemoveAll(m.Saves); err != nil {
			return err
		}
	case isLink(m.Saves):
		if err := os.Remove(m.Saves); err != nil {
			return err
		}
	}
	if m.Held != "" && exists(m.Held) {
		if exists(m.Saves) {
			return fmt.Errorf("%s is in the way of the shared saves kept at %s", m.Saves, m.Held)
		}
		if err := os.Rename(m.Held, m.Saves); err != nil {
			return err
		}
	}
	return os.RemoveAll(m.Journal)
}

// Recover finishes an unfinished swap found at journal: nothing while alive reports the game running, else Purge.
func Recover(journal string, alive func() bool) error {
	b, err := fsx.ReadFile(journalPath(journal))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if alive != nil && alive() {
		return nil
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	return Purge(m)
}
