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

const fileJournal = "file.json"

// ErrFileUnrecovered refuses a new file swap over the record of one that was never undone.
var ErrFileUnrecovered = errors.New("an earlier file swap has not been undone")

// FileManifest is the persisted record of one single-file swap: the game's file (a settings file the game rewrites
// in place) is set aside and the profile's copy takes its path.
type FileManifest struct {
	Journal string `json:"journal"`
	// Target is the game's file and Profile the profile's copy of it.
	Target  string `json:"target"`
	Profile string `json:"profile"`
	// Held is where the player's file waits, "" when the game had none.
	Held string `json:"held,omitempty"`
}

func fileJournalPath(dir string) string { return filepath.Join(dir, fileJournal) }

// HasFileJournal reports whether dir holds the record of a file swap not yet undone.
func HasFileJournal(dir string) bool { return exists(fileJournalPath(dir)) }

func persistFile(m FileManifest) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.Journal, 0o700); err != nil {
		return err
	}
	return datadir.WriteFile(fileJournalPath(m.Journal), b, 0o600)
}

// SeedFile copies the player's file to profile when the profile has none yet, so a profile starts from the player's
// current settings. A missing player's file seeds nothing.
func SeedFile(target, profile string) error {
	if exists(profile) || !exists(target) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(profile), 0o700); err != nil {
		return err
	}
	return datadir.CopyFile(target, profile)
}

// ApplyFile sets the player's file aside and puts the profile's copy at its path. The record is persisted first.
func ApplyFile(journal, target, profile string) (FileManifest, error) {
	if HasFileJournal(journal) {
		return FileManifest{}, fmt.Errorf("%w: recover %s first", ErrFileUnrecovered, journal)
	}
	m := FileManifest{Journal: journal, Target: target, Profile: profile}
	if exists(target) {
		m.Held = target + heldSuffix
		if exists(m.Held) {
			return FileManifest{}, fmt.Errorf("%s is already set aside", m.Held)
		}
	}
	if err := persistFile(m); err != nil {
		return FileManifest{}, err
	}
	if m.Held != "" {
		if err := fsx.Rename(target, m.Held); err != nil {
			_ = fsx.RemoveAll(journal)
			return FileManifest{}, err
		}
	}
	if exists(profile) {
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return m, errors.Join(err, PurgeFile(m))
		}
		if err := datadir.CopyFile(profile, target); err != nil {
			return m, errors.Join(err, PurgeFile(m))
		}
	}
	return m, nil
}

// PurgeFile writes what the game left at the path back to the profile's copy, removes it, and returns the player's
// file. Every step checks what is on disk, so it can run again after a crash.
func PurgeFile(m FileManifest) error {
	if !HasFileJournal(m.Journal) {
		return nil
	}
	if exists(m.Target) {
		if err := datadir.CopyFile(m.Target, m.Profile); err != nil {
			return err
		}
		if err := fsx.RemoveAll(m.Target); err != nil {
			return err
		}
	}
	if m.Held != "" && exists(m.Held) {
		if err := fsx.Rename(m.Held, m.Target); err != nil {
			return err
		}
	}
	return fsx.RemoveAll(m.Journal)
}

// RecoverFile finishes an unfinished file swap found at journal: nothing while alive reports the game running.
func RecoverFile(journal string, alive func() bool) error {
	b, err := fsx.ReadFile(fileJournalPath(journal))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if alive != nil && alive() {
		return nil
	}
	var m FileManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	return PurgeFile(m)
}
