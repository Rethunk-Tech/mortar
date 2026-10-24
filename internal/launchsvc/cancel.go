package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// CancelLaunch calls off the game's launch in progress: it ends the processes the launch started, then stops
// waiting for the game, and the launch ends without counting as a failure. A launch that has just come up (the
// launch screen still shows while its loader loads) is stopped as Stop would.
func (s *Service) CancelLaunch(ctx context.Context, gameID string) error {
	g, err := game.Require(gameID)
	if err != nil {
		return err
	}
	for _, sl := range s.slots(g) {
		if s.current(sl).State != Launching {
			// A game already up may still be loading its mods, which holds off a polite close until they have all
			// loaded, so a cancel ends it outright here too.
			if s.stoppable(sl) {
				return s.endSlot(sl, terminate)
			}
			continue
		}
		s.mu.Lock()
		sess, ok := s.logs[keyOf(sl)]
		cancel := s.stop[keyOf(sl)]
		handedOnly := sess.progress.Handed && sess.progress.Up == 0
		if ok && cancel != nil {
			sess.cancelled = true
			s.logs[keyOf(sl)] = sess
		}
		s.mu.Unlock()
		if !ok || cancel == nil {
			return usererr.Wrap(usererr.Busy, fmt.Errorf("%s is still getting ready to start", g.Name()))
		}
		// The game is ended first: the launch unwinds the profile from the game folder once it stops waiting. It is
		// ended outright rather than asked to close: a game still loading its mods answers a close only once they
		// have all loaded, and a launch being called off has nothing to save.
		procs, err := s.gameProcs(sl)
		var errs []error
		if err != nil {
			errs = append(errs, err)
		}
		for _, p := range procs {
			errs = append(errs, stopError(terminate(p)))
		}
		cancel()
		if handedOnly {
			go s.endLateStart(context.WithoutCancel(ctx), sl)
		}
		return errors.Join(errs...)
	}
	return usererr.Wrap(usererr.NotFound, fmt.Errorf("%s is not starting", g.Name()))
}

// cancelledRun reports whether the player called off the launch whose log is buf.
func (s *Service) cancelledRun(g game.Game, buf *launch.Buffer) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.logs[keyOf(g)]
	return sess.buf == buf && sess.cancelled
}

// lateStartWindow is how long a cancelled launch that a launcher had already accepted is watched: the launcher may
// still start the game, which would then run without the profile.
const lateStartWindow = 30 * time.Second

// endLateStart ends the game if its launcher starts it after the launch was cancelled. Only the game's own processes
// are ended (launch.Processes matches a process's program), never the launcher.
func (s *Service) endLateStart(ctx context.Context, sl slot) {
	ctx, cancel := context.WithTimeout(ctx, lateStartWindow)
	defer cancel()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("launch: %s: the launcher started no game in the %v after the launch was cancelled", sl.ID(), lateStartWindow)
			return
		case <-tick.C:
		}
		if s.current(sl).State != Idle {
			return
		}
		procs, err := s.gameProcs(sl)
		if err != nil || len(procs) == 0 {
			continue
		}
		for _, p := range procs {
			if err := terminate(p); err != nil {
				log.Printf("launch: %s: ending the game started after its launch was cancelled: %v", sl.ID(), err)
			}
		}
		log.Printf("launch: %s: ended the game its launcher started after the launch was cancelled", sl.ID())
		return
	}
}
