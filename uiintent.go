package main

import "sync"

type intent struct {
	name string
	data any
}

// UIIntentService holds window events raised while closing to the tray had destroyed the page, so the page built to
// answer them can replay them once its handlers exist.
type UIIntentService struct {
	// emit sends an event to the page.
	emit    func(name string, data any)
	mu      sync.Mutex
	pending []intent
}

// Queue keeps an event for the next page to load.
//
//wails:ignore
func (s *UIIntentService) Queue(name string, data any) {
	s.mu.Lock()
	s.pending = append(s.pending, intent{name, data})
	s.mu.Unlock()
}

// Replay emits what was queued while no page was there, once. A page calls it after registering its handlers.
func (s *UIIntentService) Replay() {
	s.mu.Lock()
	taken := s.pending
	s.pending = nil
	s.mu.Unlock()
	for _, i := range taken {
		s.emit(i.name, i.data)
	}
}
