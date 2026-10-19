package queue

// slots is how many downloads run at once: the fetch pool, which the three workers cap.
func (s *Service) slots() int {
	if s.d.Parallel != nil {
		if n := s.d.Parallel(); n >= 1 {
			return min(n, 3)
		}
	}
	return 3
}

func (s *Service) downloading() int {
	n := 0
	for _, it := range s.items {
		if it.State == StateDownloading {
			n++
		}
	}
	return n
}

// pauseLocked pauses it, keeping its partial file, and ends the transfer under way. auto marks a pause that frees a
// slot for another item, which goes on by itself later.
func (s *Service) pauseLocked(it *Item, auto bool) {
	switch it.State {
	case StateQueued, StateWaitingClick, StateDownloading:
		it.State, it.Speed, it.AutoResume = StatePaused, 0, auto
		if cancel := s.cancels[it.ID]; cancel != nil {
			cancel()
		}
	case StatePaused:
		it.AutoResume = auto && it.AutoResume
	}
}

// HoldForPlay holds the queue's downloads while a game runs that has asked for it, and lets them go on when it
// exits. Downloads under way pause as Top's preempted ones do, so they start again by themselves; items the player
// paused stay paused. Installs are not held.
//
//wails:ignore
func (s *Service) HoldForPlay(on bool) {
	s.mu.Lock()
	changed := s.playHold != on
	s.playHold = on
	if on {
		for _, it := range s.items {
			if it.State == StateDownloading {
				s.pauseLocked(it, true)
			}
		}
	}
	s.mu.Unlock()
	if changed {
		s.publish(true)
	}
	s.poke()
}

// PauseItem pauses one item: a download under way stops at once and keeps its partial file, and the rest of the
// queue goes on. It stays paused, across a restart too, until ResumeItem.
func (s *Service) PauseItem(id string) {
	s.mu.Lock()
	if it := s.find(id); it != nil {
		s.pauseLocked(it, false)
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// ResumeItem queues a paused item again; its download continues from the partial file.
func (s *Service) ResumeItem(id string) {
	s.mu.Lock()
	if it := s.find(id); it != nil && it.State == StatePaused {
		it.State, it.AutoResume = StateQueued, false
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}
