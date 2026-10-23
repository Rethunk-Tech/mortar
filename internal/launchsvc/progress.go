package launchsvc

import (
	"slices"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// ProgressEvent is emitted with the whole Progress of a launch each time it moves on.
const ProgressEvent = "launch:progress"

// Progress is how far a launch has come, for the launch screen's row of stages: the launcher it went through, the
// game's processes coming up, and the loader loading the profile's mods until it says it is done.
type Progress struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	// Via is the launcher the launch went through (launch.Command.Via); empty for a direct start.
	Via    string `json:"via"`
	Handed bool   `json:"handed"`
	// Copies is how many of the game's processes the launch starts, Up how many have come up.
	Copies int `json:"copies"`
	Up     int `json:"up"`
	// Loader is the profile's loader id, empty for a launch without mods.
	Loader string `json:"loader"`
	// Tracks is a loader whose log names each mod as it loads; Readies one whose log says when it is done.
	Tracks        bool `json:"tracks"`
	Readies       bool `json:"readies"`
	LoaderStarted bool `json:"loaderStarted"`
	// Loaded counts the mods loaded so far, Mod names the last; Total is the count to load, from the loader's own
	// line or else the profile's last run, zero when neither says.
	Loaded int    `json:"loaded"`
	Total  int    `json:"total"`
	Mod    string `json:"mod"`
	Ready  bool   `json:"ready"`
	// UsualMs is how long this profile's recent launches took to be ready, zero without any.
	UsualMs int64 `json:"usualMs"`
}

// readyChecker is a loader whose log may have no line saying it is done; one without it is assumed to have one.
type readyChecker interface{ HasReady() bool }

// usualReady is the median time to ready of the profile's last five runs that recorded one.
func usualReady(runs []Run) (ms int64, loads int) {
	var times []int64
	for _, r := range runs {
		if r.ReadyMs <= 0 {
			continue
		}
		if loads == 0 {
			loads = r.Loads
		}
		if times = append(times, r.ReadyMs); len(times) == 5 {
			break
		}
	}
	if len(times) == 0 {
		return 0, loads
	}
	slices.Sort(times)
	return times[len(times)/2], loads
}

// slowAt is when a launch that began at started counts as slow: twice its usual time to ready (at least 30 s), or
// launch.SlowAfter without a usual time.
func slowAt(started time.Time, usualMs int64) time.Time {
	if usualMs <= 0 {
		return started.Add(launch.SlowAfter)
	}
	return started.Add(max(2*time.Duration(usualMs)*time.Millisecond, 30*time.Second))
}

// newProgress is the progress of a launch that is about to start.
func (s *Service) newProgress(g game.Game, profileID string, copies int, vanilla bool) Progress {
	p := Progress{Game: g.ID(), Profile: profileID, Copies: max(copies, 1)}
	if vanilla || profileID == "" {
		return p
	}
	l, ok := s.loaderOf(g.ID(), profileID)
	if !ok {
		return p
	}
	p.Loader = l.ID()
	_, p.Tracks = l.(loader.LoadTracker)
	if _, logs := l.(loader.WithLogs); logs {
		c, checks := l.(readyChecker)
		p.Readies = !checks || c.HasReady()
	}
	if runs, err := s.Runs(g.ID(), profileID); err == nil {
		p.UsualMs, p.Total = usualReady(runs)
	}
	return p
}

// updateProgress changes the launch's progress with change and announces it, unless a newer launch replaced buf.
func (s *Service) updateProgress(g game.Game, buf *launch.Buffer, change func(*Progress)) {
	s.mu.Lock()
	sess, ok := s.logs[keyOf(g)]
	if !ok || sess.buf != buf {
		s.mu.Unlock()
		return
	}
	before := sess.progress
	change(&sess.progress)
	if sess.progress.Ready && !before.Ready {
		sess.readyMs = time.Since(sess.started).Milliseconds()
	}
	s.logs[keyOf(g)] = sess
	p := sess.progress
	s.mu.Unlock()
	if p != before {
		s.emit(ProgressEvent, p)
	}
}

// onStage moves the progress on for a stage the start reached.
func (s *Service) onStage(g game.Game, buf *launch.Buffer) func(launch.Stage) {
	return func(st launch.Stage) {
		s.updateProgress(g, buf, func(p *Progress) {
			switch st {
			case launch.StageHanded:
				p.Handed = true
			case launch.StageUp:
				p.Up = min(p.Up+1, p.Copies)
			case launch.StageLoader:
				p.LoaderStarted = true
			}
		})
	}
}

// loadLine moves the progress on for one line of the loader's log, and reports whether the line names a mod loading.
func loadLine(l loader.Loader, line string, p *Progress) bool {
	if logs, ok := l.(loader.WithLogs); ok && logs.Ready(line) {
		p.Ready = true
	}
	t, ok := l.(loader.LoadTracker)
	if !ok {
		return false
	}
	mod, total := t.Load(line)
	if total > 0 {
		p.Total = total
	}
	if mod == "" {
		return false
	}
	p.Loaded++
	p.Mod = mod
	return true
}

// LaunchProgress is the progress of the game's current or last launch.
func (s *Service) LaunchProgress(gameID string) (Progress, error) {
	g, err := game.Require(gameID)
	if err != nil {
		return Progress{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logs[keyOf(g)].progress, nil
}
