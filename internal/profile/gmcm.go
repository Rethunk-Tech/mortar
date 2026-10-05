package profile

import (
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func (s *Store) ProfileDir(game, id string) (string, error) {
	mods, err := s.ModsDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Dir(mods), nil
}

func (s *Service) GmcmMenu(game, profile string, uniqueID mod.ID) (gmcm.Capture, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Capture{}, err
	}
	return gmcm.ReadCapture(d, uniqueID)
}

func (s *Service) PendingGmcm(game, profile string, uniqueID mod.ID) (gmcm.Pending, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Pending{}, err
	}
	return gmcm.ReadPending(d, uniqueID)
}

func (s *Service) SetGmcmEdits(game, profile string, uniqueID mod.ID, edits []gmcm.Edit) error {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return err
	}
	return gmcm.WritePending(d, uniqueID, edits)
}

func (s *Service) GmcmResult(game, profile string, uniqueID mod.ID) (gmcm.Result, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Result{}, err
	}
	return gmcm.ReadResult(d, uniqueID)
}
