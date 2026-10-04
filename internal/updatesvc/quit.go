package updatesvc

import "context"

// ApplyOnQuit swaps in a staged update after Mortar exits, without reopening it. No-op when nothing is staged or Restart now already ran.
func (s *Service) ApplyOnQuit(ctx context.Context) error {
	s.lock()
	if s.info.Off != "" || s.found == nil || !s.found.Staged || s.restartChosen {
		s.mu.Unlock()
		return nil
	}
	s.restartChosen = true
	s.mu.Unlock()
	return s.u.ApplyOnExit(ctx)
}
