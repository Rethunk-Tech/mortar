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

func (s *Service) gmcmSvc() *gmcm.Service {
	return &gmcm.Service{Dirs: s.store}
}

func (s *Service) GmcmMenu(game, profile, uniqueID string) (gmcm.Capture, error) {
	return s.gmcmSvc().GmcmMenu(game, profile, uniqueID)
}

func (s *Service) PendingGmcm(game, profile, uniqueID string) (gmcm.Pending, error) {
	return s.gmcmSvc().PendingGmcm(game, profile, uniqueID)
}

func (s *Service) SetGmcmEdits(game, profile, uniqueID string, edits []gmcm.Edit) error {
	return s.gmcmSvc().SetGmcmEdits(game, profile, uniqueID, edits)
}

func (s *Service) GmcmResult(game, profile, uniqueID string) (gmcm.Result, error) {
	return s.gmcmSvc().GmcmResult(game, profile, uniqueID)
}

func (s *Service) CapturedMods(game, profile string) ([]string, error) {
	return s.gmcmSvc().CapturedMods(game, profile)
}
