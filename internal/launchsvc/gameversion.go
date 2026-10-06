package launchsvc

import (
	"context"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// askVersionFor bounds one question to the companion, which answers on localhost, so a stuck game cannot hold up
// the running-state poll that asks it.
const askVersionFor = 500 * time.Millisecond

// askGameVersion asks at the running-state poll's pace until the companion answers or the run ends.
func (s *Service) askGameVersion(ctx context.Context, g game.Game) {
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if s.noteGameVersion(ctx, g) {
				return
			}
		}
	}
}

// noteGameVersion asks the running game's companion for its version, keeps it on the session and records it as the
// version last played. It reports whether there is nothing left to ask: the version is known, the session is gone, or
// the loader's companion cannot tell it.
func (s *Service) noteGameVersion(ctx context.Context, g game.Game) bool {
	st := s.current(g)
	if st.State != Running || st.Profile == "" {
		return st.State == Idle
	}
	key := keyOf(g)
	s.mu.Lock()
	sess, ok := s.logs[key]
	s.mu.Unlock()
	if !ok || sess.gameVersion != "" || s.profiles == nil {
		return true
	}
	l, _ := s.loaderOf(g.ID(), st.Profile)
	live, ok := l.(loader.RunningGameVersion)
	if !ok {
		return true
	}
	dir, err := s.profiles.ProfileDir(g.ID(), st.Profile)
	if err != nil {
		return true
	}
	ctx, cancel := context.WithTimeout(ctx, askVersionFor)
	defer cancel()
	v, err := live.RunningGameVersion(ctx, loader.ProfileView{Game: g.ID(), Dir: dir})
	if err != nil || v == "" {
		return false
	}
	s.mu.Lock()
	sess, ok = s.logs[key]
	if ok && sess.profile == st.Profile {
		sess.gameVersion = v
		s.logs[key] = sess
	}
	s.mu.Unlock()
	if ok && s.settings != nil && !profile.IsScratch(st.Profile) {
		_, _ = s.settings.RecordLastPlayed(g.ID(), st.Profile, time.Now(), v)
	}
	return true
}
