package queue

import (
	"log"
	"time"
)

// installPoll is how often StopWait looks again for installs still under way.
const installPoll = 100 * time.Millisecond

// StopWait waits for the workers Run started once its context has ended. Downloads end with the context, but a worker
// can be stuck where none reaches (a keyring prompt), so after limit Mortar exits without it, naming the items
// still under way. An install is never cut off: it keeps writing into the store, so StopWait waits for every
// install to finish however long that takes. A download left partial keeps its resume sidecar, and a restart queues
// every item that was downloading or installing again (the loader in New).
func (s *Service) StopWait(wait func(), limit time.Duration) {
	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(limit):
	}
	logged := false
	for {
		installing, others := s.inFlight()
		if len(installing) == 0 {
			log.Printf("queue: workers still running after %s at quit, abandoned: %v", limit, others)
			return
		}
		if !logged {
			log.Printf("queue: finishing %d installs before quit", len(installing))
			logged = true
		}
		select {
		case <-done:
			return
		case <-time.After(installPoll):
		}
	}
}

// inFlight lists the ids of the items being installed and of those being downloaded.
func (s *Service) inFlight() (installing, downloading []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		switch it.State {
		case StateInstalling:
			installing = append(installing, it.ID)
		case StateDownloading:
			downloading = append(downloading, it.ID)
		}
	}
	return installing, downloading
}

// Installing reports whether any item is being installed.
func (s *Service) Installing() bool {
	installing, _ := s.inFlight()
	return len(installing) > 0
}
