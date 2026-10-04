package queue

// Active reports whether a file is downloading or installing right now; items waiting for a click or an answer do not count.
func (s *Service) Active() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		if it.State == StateDownloading || it.State == StateInstalling {
			return true
		}
	}
	return false
}
