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

func (s *Service) awaitPID(g game.Game, profileID string) {
	for {
		cur := s.current(g)
		if !cur.State.Active() {
			return
		}
		var procs []launch.Process
		var err error
		if profileID == "" {
			procs, err = s.gameProcs(g)
		} else {
			dir, dirErr := s.profiles.ModsDir(g.ID(), profileID)
			if dirErr != nil {
				return
			}
			procs, err = s.procsFor(g, dir, profileID)
		}
		if err == nil && len(procs) > 0 {
			x, waitErr := s.waitPID(procs[0].PID)
			if waitErr != nil {
				x = launch.Exit{}
			}
			s.finishWait(g, x)
			return
		}
		time.Sleep(pollEvery)
	}
}
