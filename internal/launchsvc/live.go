package launchsvc

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// LiveEvent carries a LiveRun while the game runs.
const LiveEvent = "launch:live"

// askLiveFor bounds one question to the companion, which answers on localhost, so a stuck game cannot hold up the
// next one.
const askLiveFor = 500 * time.Millisecond

// LiveRun is what the running game's companion reports, for the Running header and the mod list's loaded badges.
type LiveRun struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	Scene   string `json:"scene"`
	// Mods are the profile's packages that ship plugins, each loaded when any of its plugins is among the game's.
	Mods []LiveMod `json:"mods"`
}

type LiveMod struct {
	ID     mod.ID `json:"id"`
	Loaded bool   `json:"loaded"`
}

// followRunning asks the companion at the running-state poll's pace for as long as the run lasts, keeping the game
// version and announcing the scene and which packages loaded.
func (s *Service) followRunning(ctx context.Context, g game.Game) {
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	var packages func() map[string]startupOwner
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		st := s.current(g)
		if st.State == Idle {
			return
		}
		if st.State != Running || st.Profile == "" {
			continue
		}
		l, _ := s.loaderOf(g.ID(), st.Profile)
		live, ok := l.(loader.RunningState)
		if !ok || s.profiles == nil {
			return
		}
		if packages == nil {
			packages = s.pluginPackages(g.ID(), st.Profile)
		}
		run, ok := s.askLive(ctx, g, st, live, packages)
		if ok {
			s.emit(LiveEvent, run)
		}
	}
}

// askLive asks the companion once; the first version it reports is kept on the session and recorded as the version
// last played.
func (s *Service) askLive(ctx context.Context, g game.Game, st Status, live loader.RunningState, packages func() map[string]startupOwner) (LiveRun, bool) {
	dir, err := s.profiles.ProfileDir(g.ID(), st.Profile)
	if err != nil {
		return LiveRun{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, askLiveFor)
	defer cancel()
	got, err := live.RunningState(ctx, loader.ProfileView{Game: g.ID(), Dir: dir})
	if err != nil {
		return LiveRun{}, false
	}
	if got.GameVersion != "" {
		s.noteGameVersion(g, st.Profile, got.GameVersion)
	}
	return LiveRun{Game: g.ID(), Profile: st.Profile, Scene: got.Scene, Mods: liveMods(packages(), got.Plugins)}, true
}

func (s *Service) noteGameVersion(g game.Game, profileID, v string) {
	key := keyOf(g)
	s.mu.Lock()
	sess, ok := s.logs[key]
	fresh := ok && sess.profile == profileID && sess.gameVersion == ""
	if fresh {
		sess.gameVersion = v
		s.logs[key] = sess
	}
	s.mu.Unlock()
	if fresh && s.settings != nil && !profile.IsScratch(profileID) {
		_, _ = s.settings.RecordLastPlayed(g.ID(), profileID, time.Now(), v)
	}
}

// liveMods marks each package that ships plugins loaded when any plugin it declares is among the game's loaded ones.
func liveMods(owners map[string]startupOwner, loaded []string) []LiveMod {
	set := make(map[string]bool, len(loaded))
	for _, guid := range loaded {
		set[strings.ToLower(guid)] = true
	}
	byID := map[mod.ID]bool{}
	for guid, o := range owners {
		byID[o.ID] = byID[o.ID] || set[guid]
	}
	out := make([]LiveMod, 0, len(byID))
	for id, loaded := range byID {
		out = append(out, LiveMod{ID: id, Loaded: loaded})
	}
	slices.SortFunc(out, func(a, b LiveMod) int { return strings.Compare(string(a.ID), string(b.ID)) })
	return out
}
