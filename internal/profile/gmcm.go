package profile

import (
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/gmcm"
)

func (s *Store) ProfileDir(game, id string) (string, error) {
	mods, err := s.ModsDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Dir(mods), nil
}

func (s *Service) GmcmMenu(game, profile, uniqueID string) (gmcm.Capture, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Capture{}, err
	}
	return gmcm.ReadCapture(d, uniqueID)
}

func (s *Service) PendingGmcm(game, profile, uniqueID string) (gmcm.Pending, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Pending{}, err
	}
	return gmcm.ReadPending(d, uniqueID)
}

func (s *Service) SetGmcmEdits(game, profile, uniqueID string, edits []gmcm.Edit) error {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return err
	}
	return gmcm.WritePending(d, uniqueID, edits)
}

func (s *Service) GmcmResult(game, profile, uniqueID string) (gmcm.Result, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Result{}, err
	}
	return gmcm.ReadResult(d, uniqueID)
}

func (s *Service) CapturedMods(game, profile string) ([]string, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return nil, err
	}
	return gmcm.CapturedIDs(d)
}
