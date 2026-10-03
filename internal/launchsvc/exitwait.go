package launchsvc

import (
	"runtime"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launch"
)

func waitOnChild(req launch.Request) bool {
	if req.Direct {
		return true
	}
	return req.Vanilla && runtime.GOOS != "windows"
}

func (s *Service) armReap(gameID string) {
	s.mu.Lock()
	s.reaping[gameID] = true
	s.mu.Unlock()
}

func (s *Service) clearReap(gameID string) {
	s.mu.Lock()
	delete(s.reaping, gameID)
	delete(s.stopping, gameID)
	s.mu.Unlock()
}

func (s *Service) reapArmed(gameID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reaping[gameID]
}

func (s *Service) waitPID(pid int) (launch.Exit, error) {
	if s.WaitPID != nil {
		return s.WaitPID(pid)
	}
	return launch.WaitPID(pid)
}

func (s *Service) finishWait(g game.Game, x launch.Exit) {
	s.mu.Lock()
	if s.stopping[g.ID()] {
		x.Stopped = true
	}
	if sess, ok := s.logs[g.ID()]; ok {
		sess.exit = x
		sess.haveExit = true
		s.logs[g.ID()] = sess
	}
	s.mu.Unlock()
	cur := s.current(g.ID())
	if cur.State != Running {
		return
	}
	s.closed(g, cur, x.Stopped)
}

func (s *Service) awaitPID(g game.Game, profileID string) {
	for {
		cur := s.current(g.ID())
		if cur.State != Running && cur.State != Launching {
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
