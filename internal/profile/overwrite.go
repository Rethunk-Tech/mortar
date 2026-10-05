package profile

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// OverwriteFiles counts the files in the profile's overwrite folder, which holds what the game wrote outside its
// writable targets; 0 when there is none.
func (s *Store) OverwriteFiles(game, id string) (int, error) {
	dir, err := s.profileDir(game, id)
	if err != nil {
		return 0, err
	}
	n := 0
	err = filepath.WalkDir(filepath.Join(dir, OverwriteDir), func(_ string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type().IsRegular() {
			n++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return n, err
}

// ClearOverwrite deletes the profile's overwrite folder; a running game locks it.
func (s *Store) ClearOverwrite(game, id string) error {
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	return fsx.RemoveAll(filepath.Join(dir, OverwriteDir))
}

// OverwriteFiles counts the files the game left in the profile's overwrite folder.
func (s *Service) OverwriteFiles(game, id string) (int, error) {
	return s.store.OverwriteFiles(game, id)
}

// OpenOverwrite shows the profile's overwrite folder in the system file manager.
func (s *Service) OpenOverwrite(game, id string) error {
	dir, err := s.store.profileDir(game, id)
	if err != nil {
		return err
	}
	return datadir.Open(filepath.Join(dir, OverwriteDir))
}

// ClearOverwrite deletes what the game left in the profile's overwrite folder.
func (s *Service) ClearOverwrite(game, id string) error { return s.store.ClearOverwrite(game, id) }
