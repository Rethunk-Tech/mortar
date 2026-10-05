package profile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

// SeparateSaves reports whether the profile keeps its own saves; false when it cannot be read.
func (s *Store) SeparateSaves(gameID, id string) bool {
	p, err := s.read(gameID, id)
	return err == nil && p.SeparateSaves
}

// SavesFolder is the profile's own saves folder, which a launch swaps in for the game's shared one when the profile
// keeps its saves separate.
func (s *Store) SavesFolder(gameID, id string) (string, error) {
	dir, err := s.ProfileDir(gameID, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "saves"), nil
}

// SetSeparateSaves turns the profile's own saves folder on or off. With copyCurrent, turning it on copies the game's
// shared saves into the folder unless the folder already holds something. The shared saves are only read, and
// turning it off leaves the profile's folder where it is.
func (s *Store) SetSeparateSaves(gameID, id string, on, copyCurrent bool) (Profile, error) {
	return s.update(gameID, id, func(p *Profile, dir string) error {
		p.SeparateSaves = on
		if !on || !copyCurrent || s.settings == nil {
			return nil
		}
		own := filepath.Join(dir, "saves")
		if ents, err := os.ReadDir(own); err == nil && len(ents) > 0 {
			return nil
		}
		shared, err := game.SavesDir(s.home, s.settings.Get(), gameID, p.Install)
		if err != nil {
			return err
		}
		if _, err := os.Stat(shared); errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err := os.MkdirAll(own, 0o700); err != nil {
			return err
		}
		return datadir.CopyTree(shared, own)
	})
}
