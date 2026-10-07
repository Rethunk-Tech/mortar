package queue

import (
	"log"
	"time"
)

// waitBounded runs wait and reports whether it returned within limit; a wait that is still running is left behind.
func waitBounded(wait func(), limit time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(limit):
		return false
	}
}

// StopWait waits up to limit for the workers Run started once its context has ended. A worker can be stuck where no
// context reaches (a keyring prompt), so past the limit Mortar exits anyway, naming the items still under way. A
// download left partial keeps its resume sidecar, and a restart queues every item that was downloading or
// installing again (the loader in New).
func (s *Service) StopWait(wait func(), limit time.Duration) {
	if waitBounded(wait, limit) {
		return
	}
	s.mu.Lock()
	var names []string
	for _, it := range s.items {
		if it.State == StateDownloading || it.State == StateInstalling {
			names = append(names, it.ID+" ("+it.State+")")
		}
	}
	s.mu.Unlock()
	log.Printf("queue: workers still running after %s at quit, abandoned: %v", limit, names)
}
