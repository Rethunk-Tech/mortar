// Package launchsvc launches profiles, tracks whether the game is running one, and stops it.
package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/steam"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// StateEvent is emitted with a Status whenever a game's launch state changes.
	StateEvent = "launch:state"
	// LineEvent is emitted with a Line for each log line the game writes while a launch is waiting.
	LineEvent = "launch:line"

	pollEvery  = 2 * time.Second
	stopGrace  = 10 * time.Second
	procDirRun = "/proc"
)

// State is where a game is in its launch.
type State string

const (
	Idle      State = "idle"
	Launching State = "launching"
	Running   State = "running"
	// Failed, NoSteam and NeedsLoader are only ever emitted: the stored state returns to Idle.
	Failed      State = "failed"
	NoSteam     State = "no-steam"
	NeedsLoader State = "needs-loader"
)

// Status is a game's launch state. Profile and Since (Unix milliseconds) are set while launching or running.
type Status struct {
	Game    string      `json:"game"`
	State   State       `json:"state"`
	Profile string      `json:"profile"`
	Since   int64       `json:"since"`
	Hint    launch.Hint `json:"hint"`
	Error   string      `json:"error"`
}

// Line is one line of the game's log.
type Line struct {
	Game string `json:"game"`
	Line string `json:"line"`
}

// Service exposes launch, status and stop to the frontend.
type Service struct {
	home     string
	settings *settings.Store
	profiles *profile.Store
	procDir  string

	mu       sync.Mutex
	status   map[string]Status
	watching map[string]bool
	// App is set after application.New so events can be emitted.
	App *application.App
}

func NewService(home string, s *settings.Store, profiles *profile.Store) *Service {
	return &Service{
		home: home, settings: s, profiles: profiles, procDir: procDirRun,
		status: map[string]Status{}, watching: map[string]bool{},
	}
}

func (s *Service) emit(name string, data any) {
	if s.App != nil {
		s.App.Event.Emit(name, data)
	}
}

// set records st and announces it. Failed, NoSteam and NeedsLoader are announced but not kept.
func (s *Service) set(st Status) {
	stored := st
	switch st.State {
	case Failed, NoSteam, NeedsLoader:
		stored = Status{Game: st.Game, State: Idle}
	case Idle, Launching, Running:
	}
	s.mu.Lock()
	s.status[st.Game] = stored
	s.mu.Unlock()
	s.emit(StateEvent, st)
}

func (s *Service) current(id string) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.status[id]; ok {
		return st
	}
	return Status{Game: id, State: Idle}
}

// procsFor returns the game's loader processes that run modsDir. Where the platform gives no command line,
// the process is credited to the profile this session launched.
func (s *Service) procsFor(g game.Game, modsDir, profileID string) ([]launch.Process, error) {
	procs, err := launch.Processes(s.procDir, g.ProcessName())
	if err != nil {
		return nil, err
	}
	launched := s.current(g.ID())
	var out []launch.Process
	for _, p := range procs {
		if (p.Args == nil && launched.Profile == profileID && launched.State != Idle) || p.UsesModsPath(modsDir) {
			out = append(out, p)
		}
	}
	return out, nil
}

// Running reports whether the game is launching or running this profile, for locking its mods folder.
func (s *Service) Running(gameID, profileID string) bool {
	g := game.Find(gameID)
	if g == nil {
		return false
	}
	if cur := s.current(gameID); cur.State == Launching && cur.Profile == profileID {
		return true
	}
	dir, err := s.profiles.ModsDir(gameID, profileID)
	if err != nil {
		return false
	}
	procs, err := s.procsFor(g, dir, profileID)
	return err == nil && len(procs) > 0
}

// find returns the profile of the game the loader is running, and when that process began.
func (s *Service) find(g game.Game) (string, time.Time) {
	all, err := s.profiles.List(g.ID())
	if err != nil {
		return "", time.Time{}
	}
	for _, p := range all {
		dir, err := s.profiles.ModsDir(g.ID(), p.ID)
		if err != nil {
			continue
		}
		if procs, err := s.procsFor(g, dir, p.ID); err == nil && len(procs) > 0 {
			return p.ID, procs[0].Start
		}
	}
	return "", time.Time{}
}

func sinceOr(t time.Time) int64 {
	if t.IsZero() {
		t = time.Now()
	}
	return t.UnixMilli()
}

// poll syncs the stored state with the processes, and reports whether the game is still worth watching.
func (s *Service) poll(g game.Game) bool {
	profileID, began := s.find(g)
	cur := s.current(g.ID())
	switch {
	case cur.State == Launching:
	case cur.State == Idle && profileID != "":
		s.set(Status{Game: g.ID(), State: Running, Profile: profileID, Since: sinceOr(began)})
	case cur.State == Running && profileID == "":
		s.set(Status{Game: g.ID(), State: Idle})
	}
	return s.current(g.ID()).State != Idle
}

// watch polls every 2 s until the game is idle.
func (s *Service) watch(g game.Game) {
	s.mu.Lock()
	if s.watching[g.ID()] {
		s.mu.Unlock()
		return
	}
	s.watching[g.ID()] = true
	s.mu.Unlock()
	go func() {
		tick := time.NewTicker(pollEvery)
		defer tick.Stop()
		for range tick.C {
			if !s.poll(g) {
				s.mu.Lock()
				s.watching[g.ID()] = false
				s.mu.Unlock()
				return
			}
		}
	}()
}

// Status returns the game's launch state after looking for a game Mortar did not start.
func (s *Service) Status(gameID string) (Status, error) {
	g := game.Find(gameID)
	if g == nil {
		return Status{}, fmt.Errorf("unknown game %q", gameID)
	}
	if s.poll(g) {
		s.watch(g)
	}
	return s.current(gameID), nil
}

// Start launches the profile. Its outcome arrives as StateEvents: Launching, then Running or Failed, or NeedsLoader
// when the loader is missing or broken, or NoSteam when the user must first agree to launch without Steam (direct).
func (s *Service) Start(gameID, profileID string, direct bool) error {
	g := game.Find(gameID)
	if g == nil {
		return fmt.Errorf("unknown game %q", gameID)
	}
	if cur := s.current(gameID); cur.State == Launching || cur.State == Running {
		return fmt.Errorf("%s is already running", g.Name())
	}
	dir, err := game.InstallDir(s.home, s.settings.Get().GameFolders, gameID)
	if err != nil {
		return err
	}
	if dir == "" {
		return fmt.Errorf("%s is not installed", g.Name())
	}
	if st := g.LoaderStatus(dir, s.settings.Get().Loaders[gameID]); !st.Installed || st.Broken {
		s.set(Status{Game: gameID, State: NeedsLoader})
		return nil
	}
	if _, err := s.profiles.Mods(gameID, profileID); err != nil {
		return err
	}
	modsDir, err := s.profiles.ModsDir(gameID, profileID)
	if err != nil {
		return err
	}
	if other, _ := s.find(g); other != "" {
		return fmt.Errorf("%s is already running", g.Name())
	}
	req := launch.Request{InstallDir: dir, ModsDir: modsDir, Direct: direct}
	if st, status := steam.Locate(s.home); status == steam.Found {
		req.Steam = &st
	}
	s.set(Status{Game: gameID, State: Launching, Profile: profileID, Since: time.Now().UnixMilli()})
	s.watch(g)
	go s.run(g, profileID, req)
	return nil
}

func (s *Service) run(g game.Game, profileID string, req launch.Request) {
	err := g.Launch(context.Background(), req, func(line string) { s.emit(LineEvent, Line{Game: g.ID(), Line: line}) })
	var f *launch.Failure
	switch {
	case err == nil:
		s.set(Status{Game: g.ID(), State: Running, Profile: profileID, Since: time.Now().UnixMilli()})
	case errors.Is(err, launch.ErrNoSteam):
		s.set(Status{Game: g.ID(), State: NoSteam, Profile: profileID})
	case errors.As(err, &f):
		s.set(Status{Game: g.ID(), State: Failed, Profile: profileID, Hint: f.Hint, Error: f.Error()})
	default:
		s.set(Status{Game: g.ID(), State: Failed, Profile: profileID, Error: err.Error()})
	}
}

// Stop terminates the loader process of the profile the game is running.
func (s *Service) Stop(gameID string) error {
	g := game.Find(gameID)
	if g == nil {
		return fmt.Errorf("unknown game %q", gameID)
	}
	cur := s.current(gameID)
	if cur.State != Running {
		return fmt.Errorf("%s is not running", g.Name())
	}
	dir, err := s.profiles.ModsDir(gameID, cur.Profile)
	if err != nil {
		return err
	}
	procs, err := s.procsFor(g, dir, cur.Profile)
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range procs {
		errs = append(errs, launch.Terminate(p.PID, stopGrace))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	s.set(Status{Game: gameID, State: Idle})
	return nil
}
