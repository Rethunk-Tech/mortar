package launchsvc

import (
	"runtime"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func waitOnChild(direct, vanilla bool) bool {
	return direct || (vanilla && runtime.GOOS != "windows")
}

func (s *Service) armReap(g game.Game) {
	s.mu.Lock()
	s.reaping[keyOf(g)] = true
	s.mu.Unlock()
}

func (s *Service) clearReap(g game.Game) {
	s.mu.Lock()
	delete(s.reaping, keyOf(g))
	delete(s.stopping, keyOf(g))
	s.mu.Unlock()
}

func (s *Service) reapArmed(g game.Game) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reaping[keyOf(g)]
}

func (s *Service) waitPID(pid int) (launch.Exit, error) {
	if s.WaitPID != nil {
		return s.WaitPID(pid)
	}
	return launch.WaitPID(pid)
}

func (s *Service) finishWait(g game.Game, x launch.Exit) {
	s.mu.Lock()
	if s.stopping[keyOf(g)] {
		x.Stopped = true
	}
	if sess, ok := s.logs[keyOf(g)]; ok {
		sess.exit = x
		sess.haveExit = true
		s.logs[keyOf(g)] = sess
	}
	s.mu.Unlock()
	cur := s.current(g)
	if cur.State != Running {
		return
	}
	s.closed(g, cur, x.Stopped)
}

// awaitPID waits on the game process a launch Mortar did not start itself (a store relay) and keeps its exit for
// the run. Start refuses while any of the install's game processes run, so the first one seen is this launch's. A
// loader without process names of its own (BepInEx under Doorstop) runs inside the game, so the profile's processes
// cannot be told apart and are not searched.
func (s *Service) awaitPID(g game.Game) {
	s.mu.Lock()
	launched := s.logs[keyOf(g)].buf
	s.mu.Unlock()
	for {
		if !s.current(g).State.Active() {
			return
		}
		if procs, err := s.gameProcs(g); err == nil && len(procs) > 0 {
			x, waitErr := s.waitPID(procs[0].PID)
			if waitErr != nil {
				x = launch.Exit{}
			}
			s.mu.Lock()
			same := s.logs[keyOf(g)].buf == launched
			s.mu.Unlock()
			// A wait that outlived its run (the poll closed it) must not end the next one.
			if same {
				s.finishWait(g, x)
			}
			return
		}
		time.Sleep(pollEvery)
	}
}

// reapGrace is how long a poll that finds the game gone gives the exit waiter to report how it ended.
const reapGrace = 500 * time.Millisecond

// awaitReap waits up to reapGrace for the exit waiter to finish.
func (s *Service) awaitReap(g game.Game) {
	deadline := time.Now().Add(reapGrace)
	for s.reapArmed(g) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
}
