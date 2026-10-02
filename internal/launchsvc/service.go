// Package launchsvc launches profiles, tracks whether the game is running one, and stops it.
package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backup"
	"github.com/Rethunk-AI/mortar/internal/bridge"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/game/stardew"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/overlay"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/steam"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// StateEvent is emitted with a Status whenever a game's launch state changes.
	StateEvent = "launch:state"
	// LineEvent is emitted with a Lines for each batch of log lines the game writes, from the moment the launch
	// counts as started until the game exits.
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
	// Failed and NoSteam are only ever emitted: the stored state returns to Idle.
	Failed  State = "failed"
	NoSteam State = "no-steam"
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

// Lines is a batch of new log entries of the profile's session, oldest first.
type Lines struct {
	Game    string         `json:"game"`
	Profile string         `json:"profile"`
	Entries []launch.Entry `json:"entries"`
}

// CommandPreview is the direct launch command assembled from unsaved profile fields.
type CommandPreview struct {
	Env   []string `json:"env"`
	Argv  []string `json:"argv"`
	Error string   `json:"error"`
}

// session is the log of a launch Mortar made, and the profile it launched.
type session struct {
	buf             *launch.Buffer
	profile         string
	vanilla         bool
	started         time.Time
	restore         *settingsRestore
	settingsMissing bool
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
	// logs holds the session Mortar launched, kept after the game exits so the console can show it.
	logs map[string]session
	// stop ends the log follower of a launch when the game goes idle.
	stop map[string]context.CancelFunc
	// preparing holds the profile being readied for launch (loader check/install, then launch) and counts as running.
	preparing map[string]string
	seq       atomic.Int64
	// App is set after application.New so events can be emitted.
	App *application.App
	// EnsureLoader installs the game's loader when it is missing or broken. Start calls it before launching.
	// fromStart is true when Play requested the install, so a preparing claim for this Start must not skip it.
	EnsureLoader func(ctx context.Context, gameID string, fromStart bool) error
	// Unlocked is called when a game is no longer launching or running, so the queue can retry work it held.
	Unlocked func()
	// NotifyRunEnd sends a desktop notification when a Mortar-started run ends; main sets this from the tray wiring.
	NotifyRunEnd func(RunEndNotice)
	quit         <-chan struct{}
}

func NewService(home string, s *settings.Store, profiles *profile.Store) *Service {
	return &Service{
		home: home, settings: s, profiles: profiles, procDir: procDirRun,
		status: map[string]Status{}, watching: map[string]bool{},
		logs: map[string]session{}, stop: map[string]context.CancelFunc{}, preparing: map[string]string{},
		EnsureLoader: func(context.Context, string, bool) error { return errors.New("the loader cannot be installed here") },
	}
}

// SetLife cancels Start's loader install when ctx ends, which is when Mortar quits.
func SetLife(s *Service, ctx context.Context) {
	s.quit = ctx.Done()
}

func (s *Service) emit(name string, data any) {
	if s.App != nil {
		s.App.Event.Emit(name, data)
	}
}

// set records st and announces it. Failed and NoSteam are announced but not kept.
func (s *Service) set(st Status) {
	stored := st
	switch st.State {
	case Failed, NoSteam:
		stored = Status{Game: st.Game, State: Idle}
	case Idle, Launching, Running:
	}
	s.mu.Lock()
	prev, had := s.status[st.Game]
	s.status[st.Game] = stored
	if stored.State == Idle && s.stop[st.Game] != nil {
		s.stop[st.Game]()
		delete(s.stop, st.Game)
	}
	s.mu.Unlock()
	s.emit(StateEvent, st)
	if st.State == Running && st.Profile != "" && s.settings != nil {
		_, _ = s.settings.RecordLastPlayed(st.Game, st.Profile, time.Now(), launchGameVersion(s, st.Game))
	}
	if stored.State == Idle && had && prev.State != Idle && s.Unlocked != nil {
		s.Unlocked()
	}
}

func launchGameVersion(s *Service, gameID string) string {
	s.mu.Lock()
	sess := s.logs[gameID]
	s.mu.Unlock()
	if sess.buf == nil {
		return ""
	}
	for _, e := range sess.buf.Lines() {
		if v := stardew.StardewVersionFromLog(e.Message); v != "" {
			return v
		}
	}
	return ""
}

func (s *Service) current(id string) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.status[id]; ok {
		return st
	}
	return Status{Game: id, State: Idle}
}

// procsFor returns the game's loader processes that run modsDir.
func (s *Service) procsFor(g game.Game, modsDir, profileID string) ([]launch.Process, error) {
	procs, err := launch.Processes(s.procDir, g.ProcessName())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	cur := s.status[g.ID()]
	sess, ok := s.logs[g.ID()]
	s.mu.Unlock()
	launched := ""
	vanilla := false
	if ok && cur.State != Idle {
		launched = sess.profile
		vanilla = sess.vanilla
	}
	var out []launch.Process
	for _, p := range procs {
		if credited(p, modsDir, profileID, launched, vanilla) {
			out = append(out, p)
		}
	}
	return out, nil
}

// credited reports whether the loader process p runs profileID, whose mods folder is modsDir. launched is the
// profile of the launch Mortar made and has not seen end, if any. Where the platform gives no command line (Windows),
// the process is credited to that profile, or to every profile when Mortar did not start it: it may run any of them.
func credited(p launch.Process, modsDir, profileID, launched string, vanilla bool) bool {
	if vanilla {
		return false
	}
	if p.Args == nil {
		return launched == "" || launched == profileID
	}
	return p.UsesModsPath(modsDir)
}

// Running reports whether the game is launching or running this profile, for locking its mods folder.
func (s *Service) Running(gameID, profileID string) bool {
	g := game.Find(gameID)
	if g == nil {
		return false
	}
	s.mu.Lock()
	cur, prep := s.status[gameID], s.preparing[gameID]
	s.mu.Unlock()
	if (prep != "" && prep == profileID) || (cur.State == Launching && cur.Profile != "" && cur.Profile == profileID) {
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
func (s *Service) gameProcs(g game.Game) ([]launch.Process, error) {
	var out []launch.Process
	for _, name := range g.GameProcesses() {
		ps, err := launch.Processes(s.procDir, name)
		if err != nil {
			return nil, err
		}
		out = append(out, ps...)
	}
	return out, nil
}

func (s *Service) seen(g game.Game) func() bool {
	return func() bool {
		ps, err := s.gameProcs(g)
		return err == nil && len(ps) > 0
	}
}

func (s *Service) poll(g game.Game) bool {
	profileID, began := s.find(g)
	procs, _ := s.gameProcs(g)
	alive := len(procs) > 0
	cur := s.current(g.ID())
	switch {
	case cur.State == Launching:
	case cur.State == Idle && profileID != "":
		// A game Mortar did not start has its own log; the last session's buffer is stale.
		s.mu.Lock()
		delete(s.logs, g.ID())
		s.mu.Unlock()
		s.set(Status{Game: g.ID(), State: Running, Profile: profileID, Since: sinceOr(began)})
	case cur.State == Idle && alive:
		s.mu.Lock()
		delete(s.logs, g.ID())
		s.mu.Unlock()
		s.set(Status{Game: g.ID(), State: Running, Since: sinceOr(procs[0].Start)})
	case cur.State == Running && cur.Profile != "" && profileID == "":
		s.closed(g, cur, false)
	case cur.State == Running && cur.Profile == "" && !alive:
		s.closed(g, cur, false)
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

// Start launches the profile. Its outcome arrives as StateEvents: Launching, then Running or Failed, or NoSteam
// when the user must first agree to launch without Steam (direct). A missing or broken loader is installed first.
func (s *Service) Start(ctx context.Context, gameID, profileID string, direct bool) error {
	// The game outlives the call that started it, so the caller's cancellation does not reach the launch.
	return s.start(context.WithoutCancel(ctx), gameID, profileID, direct)
}

func (s *Service) start(parent context.Context, gameID, profileID string, direct bool) error {
	g := game.Find(gameID)
	if g == nil {
		return fmt.Errorf("unknown game %q", gameID)
	}
	// preparing is claimed under the same lock as the check, so two Starts cannot both pass it.
	s.mu.Lock()
	cur := s.status[gameID]
	_, busy := s.preparing[gameID]
	busy = busy || cur.State == Launching || cur.State == Running
	if !busy {
		s.preparing[gameID] = profileID
	}
	s.mu.Unlock()
	if busy {
		return fmt.Errorf("%s is already running", g.Name())
	}
	dir, modsDir, err := s.target(g, profileID)
	if err != nil {
		s.donePreparing(gameID)
		return err
	}
	// Even an installed loader goes through EnsureLoader: it waits out an update in progress, which would
	// otherwise launch the game on half-replaced files.
	go func() {
		defer s.donePreparing(gameID)
		ctx, cancel := context.WithCancel(parent)
		defer cancel()
		// Checked before the watcher starts, so a quit that already happened cancels before EnsureLoader runs.
		select {
		case <-s.quit:
			cancel()
		default:
			go func() {
				select {
				case <-s.quit:
					cancel()
				case <-ctx.Done():
				}
			}()
		}
		err := s.EnsureLoader(ctx, gameID, true)
		if err != nil {
			err = fmt.Errorf("could not install %s: %w", g.LoaderName(), err)
		} else {
			// Reading the profile takes its lock, so a change to its mods already under way finishes first; any
			// later one sees the profile as running.
			if _, err = s.profiles.Mods(gameID, profileID); err == nil {
				err = s.begin(ctx, g, profileID, dir, modsDir, direct, false)
			}
		}
		if err != nil {
			s.set(Status{Game: gameID, State: Failed, Profile: profileID, Error: err.Error()})
		}
	}()
	return nil
}

// ForcesSMAPI reports whether Steam's launch options will start SMAPI even for a vanilla launch.
func (s *Service) ForcesSMAPI(gameID string) (bool, error) {
	g := game.Find(gameID)
	if g == nil {
		return false, fmt.Errorf("unknown game %q", gameID)
	}
	if runtime.GOOS != "windows" {
		return false, nil
	}
	st, status := steam.Locate(s.home)
	if status != steam.Found {
		return false, nil
	}
	opts, err := st.LaunchOptions(g.SteamAppID())
	if err != nil {
		return false, err
	}
	return g.SteamLaunchForcesLoader(opts), nil
}

// StartVanilla launches the game without a profile mods folder. On Windows that is steam -applaunch
// with no extra arguments (Steam launch options may still force SMAPI). On Linux SMAPI replaced the
// game launcher, so Mortar starts StardewValley-original directly.
func (s *Service) StartVanilla(gameID string, direct bool) error {
	g := game.Find(gameID)
	if g == nil {
		return fmt.Errorf("unknown game %q", gameID)
	}
	s.mu.Lock()
	cur := s.status[gameID]
	_, busy := s.preparing[gameID]
	busy = busy || cur.State == Launching || cur.State == Running
	if !busy {
		s.preparing[gameID] = ""
	}
	s.mu.Unlock()
	if busy {
		return fmt.Errorf("%s is already running", g.Name())
	}
	dir, err := game.InstallDir(s.home, s.settings.Get(), g.ID())
	if err != nil {
		s.donePreparing(gameID)
		return err
	}
	if dir == "" {
		s.donePreparing(gameID)
		return fmt.Errorf("%s is not installed", g.Name())
	}
	go func() {
		defer s.donePreparing(gameID)
		if err := s.begin(context.Background(), g, "", dir, "", direct, true); err != nil {
			s.set(Status{Game: gameID, State: Failed, Error: err.Error()})
		}
	}()
	return nil
}

// target returns the game's install folder and the profile's mods folder.
func (s *Service) target(g game.Game, profileID string) (dir, modsDir string, err error) {
	dir, err = game.InstallDir(s.home, s.settings.Get(), g.ID())
	if err != nil {
		return "", "", err
	}
	if dir == "" {
		return "", "", fmt.Errorf("%s is not installed", g.Name())
	}
	if _, err := s.profiles.Mods(g.ID(), profileID); err != nil {
		return "", "", err
	}
	modsDir, err = s.profiles.ModsDir(g.ID(), profileID)
	return dir, modsDir, err
}

// PreviewCommand builds the direct launch command without starting it.
func (s *Service) PreviewCommand(gameID, profileID, options, prefix, env string) CommandPreview {
	preview := CommandPreview{}
	fail := func(err error) CommandPreview {
		preview.Error = err.Error()
		return preview
	}
	g := game.Find(gameID)
	if g == nil {
		return fail(fmt.Errorf("unknown game %q", gameID))
	}
	dir, modsDir, err := s.target(g, profileID)
	if err != nil {
		return fail(err)
	}
	extra, err := stardew.ParseLaunchOptions(options)
	if err != nil {
		return fail(err)
	}
	prefixArgs, err := profile.LaunchPrefixArgs(prefix)
	if err != nil {
		return fail(err)
	}
	envArgs, err := profile.LaunchEnvironment(env)
	if err != nil {
		return fail(err)
	}
	builder, ok := g.(interface {
		DirectCommand(string, launch.Request) (launch.Command, error)
	})
	if !ok {
		return fail(fmt.Errorf("%s does not support command previews", g.Name()))
	}
	cmd, err := builder.DirectCommand(runtime.GOOS, launch.Request{
		InstallDir: dir,
		ModsDir:    modsDir,
		ExtraArgs:  extra,
		Prefix:     prefixArgs,
		Env:        envArgs,
	})
	if err != nil {
		return fail(err)
	}
	preview.Env = cmd.Env
	preview.Argv = append([]string{cmd.Name}, cmd.Args...)
	return preview
}

func (s *Service) donePreparing(gameID string) {
	s.mu.Lock()
	delete(s.preparing, gameID)
	st := s.status[gameID]
	s.mu.Unlock()
	if st.State != Launching && st.State != Running && s.Unlocked != nil {
		s.Unlocked()
	}
}

// begin starts the launch once the loader is in place.
func (s *Service) begin(ctx context.Context, g game.Game, profileID, dir, modsDir string, direct, vanilla bool) error {
	gameID := g.ID()
	if ps, err := s.gameProcs(g); err == nil && len(ps) > 0 {
		return fmt.Errorf("%s is already running", g.Name())
	}
	req := launch.Request{InstallDir: dir, ModsDir: modsDir, Direct: direct, Vanilla: vanilla, Seen: s.seen(g)}
	if !vanilla && profileID != "" {
		st := s.settings.Get()
		if err := overlay.ApplyToMods(modsDir, st.OverlayEnabled, st.OverlayPort, st.OverlayToken); err != nil {
			return err
		}
		opts, err := s.profiles.LaunchOptions(g.ID(), profileID)
		if err != nil {
			return err
		}
		extra, err := stardew.ParseLaunchOptions(opts)
		if err != nil {
			return err
		}
		req.ExtraArgs = extra
		prefix, env, err := s.profiles.LaunchSettings(g.ID(), profileID)
		if err != nil {
			return err
		}
		req.Prefix, err = profile.LaunchPrefixArgs(prefix)
		if err != nil {
			return err
		}
		req.Env, err = profile.LaunchEnvironment(env)
		if err != nil {
			return err
		}
	}
	cur := s.settings.Get()
	_, store, _, err := game.Resolve(s.home, cur, g.ID())
	if err != nil {
		return err
	}
	if store == game.StoreGOG || store == game.StoreGOGHeroic || store == game.StoreMinigalaxy || store == game.StoreLutris {
		req.Direct = true
	}
	if st, status := steam.Locate(s.home); status == steam.Found {
		req.Steam = &st
	}
	if store == game.StoreFlatpakSteam {
		for _, one := range steam.LocateAll(s.home) {
			if one.Kind == steam.KindFlatpak {
				req.Steam = &one
				break
			}
		}
	}
	var restore *settingsRestore
	var settingsMissing bool
	if !vanilla && profileID != "" {
		restore, settingsMissing, err = s.prepareGameSettings(g.ID(), profileID)
		if err != nil {
			return err
		}
		if err := s.backupChangedSaves(g.ID(), profileID); err != nil {
			log.Printf("save backup before launch: %v", err)
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	buf := &launch.Buffer{}
	started := time.Now()
	s.mu.Lock()
	s.logs[gameID], s.stop[gameID] = session{
		buf:             buf,
		profile:         profileID,
		vanilla:         vanilla,
		started:         started,
		restore:         restore,
		settingsMissing: settingsMissing,
	}, cancel
	s.mu.Unlock()
	s.set(Status{Game: gameID, State: Launching, Profile: profileID, Since: started.UnixMilli()})
	if settingsMissing {
		s.say(gameID, profileID, "startup_preferences is missing; skipped profile game settings.")
	}
	s.watch(g)
	go s.run(runCtx, g, profileID, req, buf)
	return nil
}

func (s *Service) backupChangedSaves(gameID, profileID string) error {
	events, err := s.profiles.History(gameID, profileID)
	if err != nil {
		return err
	}
	runs, err := s.Runs(gameID, profileID)
	if err != nil {
		return err
	}
	var lastRun time.Time
	if len(runs) > 0 {
		lastRun, err = time.Parse(time.RFC3339Nano, runs[0].Started)
		if err != nil {
			return err
		}
	}
	if !changedSinceLastRun(events, lastRun) {
		return nil
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	base, err := datadir.Dir()
	if err != nil {
		return err
	}
	_, err = backup.Saves(
		filepath.Join(cfg, "StardewValley", "Saves"),
		filepath.Join(base, "backups"),
		s.settings.Get().BackupsKept,
		time.Now(),
		backup.Cause{Profile: profileID, Kind: backup.KindUpdate},
	)
	return err
}

func changedSinceLastRun(events []profile.HistoryEvent, lastRun time.Time) bool {
	for _, event := range events {
		if event.At.After(lastRun) {
			return true
		}
	}
	return false
}

// collect parses log lines into buf and announces them, unless a newer launch has replaced buf.
func (s *Service) collect(gameID, profileID string, buf *launch.Buffer) func([]string) {
	var p launch.Parser
	return func(lines []string) {
		entries := make([]launch.Entry, 0, len(lines))
		for _, l := range lines {
			if e, shown := p.Parse(l); shown {
				e.Seq = s.seq.Add(1)
				entries = append(entries, e)
			}
		}
		s.mu.Lock()
		current := s.logs[gameID].buf == buf
		s.mu.Unlock()
		if !current {
			return
		}
		for _, e := range entries {
			buf.Add(e)
		}
		s.emit(LineEvent, Lines{Game: gameID, Profile: profileID, Entries: entries})
	}
}

// Lines returns the profile's log, oldest first, at most launch.MaxLines: this session's when Mortar launched the
// game, else the log file on disk, which is what the last session left after a crash. It is empty when the log is
// another profile's.
func (s *Service) Lines(gameID, profileID string) ([]launch.Entry, error) {
	g := game.Find(gameID)
	if g == nil {
		return nil, fmt.Errorf("unknown game %q", gameID)
	}
	s.mu.Lock()
	sess, ok := s.logs[gameID]
	s.mu.Unlock()
	if ok {
		if sess.vanilla {
			if profileID == "" {
				return sess.buf.Lines(), nil
			}
			return []launch.Entry{}, nil
		}
		if sess.profile == profileID {
			return sess.buf.Lines(), nil
		}
		return []launch.Entry{}, nil
	}
	path, err := g.LogFile()
	if err != nil {
		return nil, err
	}
	data, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []launch.Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	modsDir, err := s.profiles.ModsDir(gameID, profileID)
	if err != nil {
		return nil, err
	}
	text := strings.ToValidUTF8(string(data), "")
	if !launch.LogOwnedBy(text, s.home, modsDir) {
		return []launch.Entry{}, nil
	}
	var p launch.Parser
	var out launch.Buffer
	for l := range strings.SplitSeq(text, "\n") {
		if l = strings.TrimRight(l, "\r"); l != "" {
			if e, shown := p.Parse(l); shown {
				e.Seq = s.seq.Add(1)
				out.Add(e)
			}
		}
	}
	return out.Lines(), nil
}

func (s *Service) run(ctx context.Context, g game.Game, profileID string, req launch.Request, buf *launch.Buffer) {
	err := g.Launch(ctx, req, s.collect(g.ID(), profileID, buf))
	if err != nil {
		s.mu.Lock()
		sess := s.logs[g.ID()]
		s.mu.Unlock()
		if sess.buf == buf && sess.profile == profileID {
			s.restoreGameSettings(sess.restore)
		}
	}
	var f *launch.Failure
	var exited *launch.ExitError
	switch {
	case err == nil:
		s.set(Status{Game: g.ID(), State: Running, Profile: profileID, Since: time.Now().UnixMilli()})
		if req.Vanilla {
			s.say(g.ID(), profileID, "Started without mods")
		}
	case errors.Is(err, launch.ErrNoSteam):
		s.set(Status{Game: g.ID(), State: NoSteam, Profile: profileID})
	case errors.As(err, &exited):
		if len(buf.Lines()) == 0 {
			s.say(g.ID(), profileID, fmt.Sprintf("%s exited with code %d.", g.LoaderName(), exited.Code))
		}
		s.finishFailed(g, profileID, buf)
		s.set(Status{Game: g.ID(), State: Failed, Profile: profileID, Error: err.Error()})
	case errors.As(err, &f):
		s.finishFailed(g, profileID, buf)
		s.set(Status{Game: g.ID(), State: Failed, Profile: profileID, Hint: f.Hint, Error: f.Error()})
	default:
		s.finishFailed(g, profileID, buf)
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
	var procs []launch.Process
	var err error
	if cur.Profile == "" {
		procs, err = s.gameProcs(g)
	} else {
		var dir string
		dir, err = s.profiles.ModsDir(gameID, cur.Profile)
		if err != nil {
			return err
		}
		procs, err = s.procsFor(g, dir, cur.Profile)
	}
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
	s.closed(g, cur, true)
	return nil
}

// closed ends the game's session with a console line of Mortar's own, so the log does not just stop, then marks
// the game idle. It does nothing when a concurrent poll or Stop already closed the session.
func (s *Service) closed(g game.Game, cur Status, stopped bool) {
	if s.current(g.ID()).State != Running {
		return
	}
	msg := g.Name() + " closed"
	if stopped {
		msg = g.Name() + " was stopped from Mortar"
	}
	if cur.Since > 0 {
		msg += " after " + time.Since(time.UnixMilli(cur.Since)).Round(time.Second).String()
	}
	s.say(g.ID(), cur.Profile, msg+".")
	s.mu.Lock()
	sess, ok := s.logs[g.ID()]
	s.mu.Unlock()
	if ok && !sess.vanilla && cur.Profile != "" {
		s.restoreGameSettings(sess.restore)
		started := sess.started
		if cur.Since > 0 {
			started = time.UnixMilli(cur.Since)
		}
		if s.NotifyRunEnd != nil {
			modsDir, err := s.profiles.ModsDir(g.ID(), cur.Profile)
			if err == nil {
				stats := launch.Summarize(s.runText(g, cur.Profile, modsDir))
				title, body := RunEndNotificationText(g.Name(), stats)
				s.NotifyRunEnd(RunEndNotice{Game: g.ID(), Profile: cur.Profile, Title: title, Body: body})
			}
		}
		s.record(g, cur.Profile, started, false)
	}
	s.set(Status{Game: g.ID(), State: Idle})
}

func (s *Service) finishFailed(g game.Game, profileID string, buf *launch.Buffer) {
	s.mu.Lock()
	sess, ok := s.logs[g.ID()]
	s.mu.Unlock()
	if !ok || sess.vanilla || sess.buf != buf || profileID == "" {
		return
	}
	s.record(g, profileID, sess.started, true)
}

// say adds a console line of Mortar's own to the session of the profile the game runs.
func (s *Service) say(gameID, profileID, msg string) {
	e := launch.Entry{Seq: s.seq.Add(1), Time: time.Now().Format(time.TimeOnly), Level: launch.Info, Mod: "Mortar", Message: msg}
	s.mu.Lock()
	sess, ok := s.logs[gameID]
	s.mu.Unlock()
	if ok && sess.profile == profileID {
		sess.buf.Add(e)
	}
	s.emit(LineEvent, Lines{Game: gameID, Profile: profileID, Entries: []launch.Entry{e}})
}

// Send runs a console command in the running game through the bridge mod in the running profile. The command
// is echoed into the console; its output arrives with the game's own log.
func (s *Service) Send(gameID, command string) error {
	g := game.Find(gameID)
	if g == nil {
		return fmt.Errorf("unknown game %q", gameID)
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return errors.New("enter a command")
	}
	cur := s.current(gameID)
	if cur.State != Running {
		return fmt.Errorf("%s is not running", g.Name())
	}
	folder, err := s.bridgeFolder(g, cur.Profile)
	if err != nil {
		return err
	}
	if err := bridge.Send(folder, command); err != nil {
		return err
	}
	s.say(gameID, cur.Profile, "> "+command)
	return nil
}

// bridgeFolder is the bridge mod's folder in the profile the game runs, under the key that profile holds: a
// Mortar update can bundle a newer bridge than the one the running game loaded.
func (s *Service) bridgeFolder(g game.Game, profileID string) (string, error) {
	all, err := s.profiles.List(g.ID())
	if err != nil {
		return "", err
	}
	for _, p := range all {
		if p.ID != profileID {
			continue
		}
		for _, e := range p.Entries {
			if e.Source.Kind == profile.SourceMortar {
				modsDir, err := s.profiles.ModsDir(g.ID(), profileID)
				if err != nil {
					return "", err
				}
				return filepath.Join(modsDir, e.Key, bridge.ModFolder), nil
			}
		}
	}
	return "", fmt.Errorf("%s has no console bridge in this profile", g.Name())
}
