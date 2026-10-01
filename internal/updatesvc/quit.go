package updatesvc

import "context"

// ShouldApplyOnQuit is true when a verified update is staged and the user did not already choose Restart now.
func (s *Service) ShouldApplyOnQuit() bool {
	if s.info.Off != "" {
		return false
	}
	s.lock()
	defer s.mu.Unlock()
	return s.found != nil && s.found.Staged && !s.restartChosen
}

// ApplyOnQuit swaps in a staged update as Mortar exits. No-op when ShouldApplyOnQuit is false.
func (s *Service) ApplyOnQuit(ctx context.Context) error {
	s.lock()
	if s.info.Off != "" || s.found == nil || !s.found.Staged || s.restartChosen {
		s.mu.Unlock()
		return nil
	}
	s.restartChosen = true
	s.mu.Unlock()
	return s.u.Restart(ctx)
}
