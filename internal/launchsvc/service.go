// Package launchsvc launches profiles, tracks whether the game is running one, and stops it.
package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/overlay"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/steam"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

// pollEvery is how often a running game is checked; tests set it small.
var pollEvery = 2 * time.Second

// procVisible, when set, limits which of the live processes count as a game; tests use it so a game running outside the test, such
// as another test binary's stand-in, is never mistaken for their own.
var procVisible func(launch.Process) bool

const (
	// StateEvent is emitted with a Status whenever a game's launch state changes.
	StateEvent = "launch:state"
	// LineEvent is emitted with a Lines for each batch of log lines the game writes, from the moment the launch
	// counts as started until the game exits.
	LineEvent = "launch:line"
	// BackupWarningEvent is emitted when a launch could not make its pre-play save backup.
	BackupWarningEvent = "launch:backup-warning"
	// SettingsRestoreWarningEvent is emitted when profile game settings could not be restored.
	SettingsRestoreWarningEvent = "launch:settings-restore-warning"

	stopGrace  = 10 * time.Second
	procDirRun = "/proc"
)

// State is where a game is in its launch.
type State string

// Active reports whether the game is starting or running.
func (s State) Active() bool { return s == Launching || s == Running }

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
	Game    string `json:"game"`
	State   State  `json:"state"`
	Profile string `json:"profile"`
	// Install is the id of the game install the launch runs; a game Mortar did not start is credited to the selected install.
	Install string      `json:"install,omitempty"`
	Since   int64       `json:"since"`
	Hint    launch.Hint `json:"hint"`
	Error   string      `json:"error"`
	Cause   *Cause      `json:"cause,omitempty"`
}

// Lines is a batch of new log entries of the profile's session, oldest first.
type Lines struct {
	Game    string         `json:"game"`
	Profile string         `json:"profile"`
	Entries []launch.Entry `json:"entries"`
}

// BackupWarning describes a non-fatal save backup failure before Play.
type BackupWarning struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	Error   string `json:"error"`
}

// SettingsRestoreWarning describes a profile game settings restore failure.
type SettingsRestoreWarning struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	Error   string `json:"error"`
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
	preset          string
	vanilla         bool
	started         time.Time
	mods            []launch.ModRef
	restore         *settingsRestore
	deployed        *deployment
	settingsMissing bool
	exit            launch.Exit
	haveExit        bool
	// stoppedAt is when the player asked Mortar to stop the game; zero for any other end.
	stoppedAt time.Time
	// gameVersion is what the running game's companion reported, for a loader whose log names no version.
	gameVersion string
}

// Service exposes launch, status and stop to the frontend.
type Service struct {
	home     string
	settings *settings.Store
	profiles *profile.Store
	procDir  string
	dataDir  func() (string, error)

	mu       sync.Mutex
	status   map[string]Status
	watching map[string]bool
	// logs holds the session Mortar launched, kept after the game exits so the console can show it.
	logs map[string]session
	// stop ends the log follower of a launch when the game goes idle.
	stop map[string]context.CancelFunc
	// preparing holds the profile being readied for launch (loader check/install, then launch) and counts as running.
	preparing map[string]string
	// startFailed is the error of the last start of each slot that failed, until the next start; a failed state is not
	// kept in status, so a caller that waits on a launch reads it here.
	startFailed map[string]string
	// closing marks a run whose end is being recorded, so a second exit report for it is ignored.
	closing map[string]bool
	// lastFailure is why the last launch of each slot failed, from any path, until the next start: the failed state is
	// not kept in status, and the run record and a waiting caller read the cause here.
	lastFailure map[string]string
	seq         atomic.Int64
	// App is set after application.New so events can be emitted.
	App winhost.Host
	// EnsureLoader installs the game's loader when it is missing or broken. Start calls it before launching.
	// fromStart is true when Play requested the install, so a preparing claim for this Start must not skip it.
	EnsureLoader func(ctx context.Context, gameID, loaderID string, fromStart bool) error
	// Unlocked is called when a game is no longer launching or running, so the queue can retry work it held.
	Unlocked func()
	// NotifyRunEnd sends a desktop notification when a run Mortar started crashed; main sets this from the tray wiring.
	NotifyRunEnd func(RunEndNotice)
	// OnSavePlayed is called, when a run is recorded, with each save the run played: the one SMAPI loaded, or the
	// catalog saves written during the run.
	OnSavePlayed func(gameID, profileID, saveFolder string)
	quit         <-chan struct{}
	// Starter starts launch plans; tests replace its runner and lookups.
	Starter game.Starter
	// WaitPID waits for a game process Mortar did not start (Steam relay). Tests replace it.
	WaitPID  func(pid int) (launch.Exit, error)
	stopping map[string]bool
	reaping  map[string]bool
	// sampled is closed once a measured launch's sampler has stopped its session; the game must still be running then.
	sampled map[string]chan struct{}
	// SweepVersions, SweepCompat, SweepHasUpdate and SweepMissingDeps are replaced in tests.
	SweepVersions    func(gameID string) (gameVer, smapiVer string, err error)
	SweepCompat      func(ctx context.Context) (meta.CompatIndex, error)
	SweepHasUpdate   func(uniqueID mod.ID, nexusID int) bool
	SweepMissingDeps func(gameID, profileID string) int
}

func NewService(home string, s *settings.Store, profiles *profile.Store) *Service {
	return &Service{
		home: home, settings: s, profiles: profiles, procDir: procDirRun, dataDir: datadir.Dir,
		status: map[string]Status{}, watching: map[string]bool{},
		logs: map[string]session{}, stop: map[string]context.CancelFunc{}, preparing: map[string]string{}, startFailed: map[string]string{}, closing: map[string]bool{}, lastFailure: map[string]string{},
		stopping: map[string]bool{}, reaping: map[string]bool{}, sampled: map[string]chan struct{}{},
		EnsureLoader: func(context.Context, string, string, bool) error {
			return errors.New("the loader cannot be installed here")
		},
	}
}

// SetLife cancels Start's loader install when ctx ends, which is when Mortar quits.
func SetLife(s *Service, ctx context.Context) {
	s.quit = ctx.Done()
}

func (s *Service) emit(name string, data any) {
	if s.App != nil {
		s.App.Emit(name, data)
	}
}

// failStart records that the game did not start, then sets st. A loader that exits before it is ready is not recorded:
// the run's own summary says how it ended.
func (s *Service) failStart(st Status) {
	s.mu.Lock()
	s.startFailed[slotKey(st.Game, st.Install)] = st.Error
	s.mu.Unlock()
	s.set(st)
}

// set records st and announces it. Failed and NoSteam are announced but not kept.
func (s *Service) set(st Status) {
	key := slotKey(st.Game, st.Install)
	stored := st
	switch st.State {
	case Failed, NoSteam:
		if st.State == Failed {
			log.Printf("launch: %s %s failed: %s", st.Game, st.Profile, st.Error)
		}
		stored = Status{Game: st.Game, Install: st.Install, State: Idle}
	case Launching, Running:
		log.Printf("launch: %s %s %s", st.Game, st.Profile, st.State)
	case Idle:
	}
	s.mu.Lock()
	if stored.State == Idle {
		delete(s.closing, key)
	}
	if st.State == Launching {
		delete(s.startFailed, key)
		delete(s.lastFailure, key)
	}
	prev, had := s.status[key]
	s.status[key] = stored
	if stored.State == Idle && s.stop[key] != nil {
		s.stop[key]()
		delete(s.stop, key)
	}
	s.mu.Unlock()
	s.emit(StateEvent, st)
	if st.State == Running && st.Profile != "" && !profile.IsScratch(st.Profile) && s.settings != nil {
		_, _ = s.settings.RecordLastPlayed(st.Game, st.Profile, time.Now(), launchGameVersion(s, key))
	}
	if stored.State == Idle && had && prev.State != Idle && s.Unlocked != nil {
		s.Unlocked()
	}
}

func launchGameVersion(s *Service, key string) string {
	s.mu.Lock()
	sess := s.logs[key]
	s.mu.Unlock()
	if sess.buf == nil {
		return ""
	}
	gameID, _, _ := strings.Cut(key, "/")
	l, ok := game.PrimaryLoader(gameID)
	if !ok {
		return ""
	}
	versions, ok := l.(loader.GameVersion)
	if !ok {
		return ""
	}
	for _, e := range sess.buf.Lines() {
		if v := versions.GameVersion(e.Message); v != "" {
			return v
		}
	}
	return ""
}

func (s *Service) current(g game.Game) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.status[keyOf(g)]; ok {
		return st
	}
	return Status{Game: g.ID(), Install: installOf(g), State: Idle}
}

// procsFor returns the loader processes of g's install that run modsDir.
func (s *Service) procsFor(g game.Game, modsDir, profileID string) ([]launch.Process, error) {
	l, _ := s.loaderOf(g.ID(), profileID)
	// A loader without executables of its own runs inside the game's process.
	names := g.GameProcesses()
	if own, ok := l.(loader.ProcessNames); ok {
		names = own.ProcessNames()
	}
	procs, err := launch.Processes(s.procDir, names...)
	if err != nil {
		return nil, err
	}
	procs = s.ownedBy(g, procs)
	owner, _ := l.(loader.Owner)
	s.mu.Lock()
	cur := s.status[keyOf(g)]
	sess, ok := s.logs[keyOf(g)]
	s.mu.Unlock()
	launched := ""
	vanilla := false
	if ok && cur.State != Idle {
		launched = sess.profile
		vanilla = sess.vanilla
	}
	var out []launch.Process
	for _, p := range procs {
		if credited(owner, p, modsDir, profileID, launched, vanilla) {
			out = append(out, p)
		}
	}
	return out, nil
}

// credited reports whether the loader process p runs profileID, whose mods folder is modsDir. launched is the
// profile of the launch Mortar made and has not seen end, if any. Where the platform gives no command line (Windows),
// the process is credited to that profile, or to every profile when Mortar did not start it: it may run any of them.
func credited(owner loader.Owner, p launch.Process, modsDir, profileID, launched string, vanilla bool) bool {
	if vanilla {
		return false
	}
	if p.Args == nil || owner == nil {
		return launched == "" || launched == profileID
	}
	return owner.Owns(loader.Process{PID: p.PID, Args: p.Args}, loader.ProfileView{Dir: filepath.Dir(modsDir)})
}

// Running reports whether the game is launching or running this profile, for locking its mods folder.
func (s *Service) Running(gameID, profileID string) bool {
	g := game.Find(gameID)
	if g == nil {
		return false
	}
	dir, dirErr := s.profiles.ModsDir(gameID, profileID)
	for _, sl := range s.slots(g) {
		s.mu.Lock()
		cur, prep := s.status[keyOf(sl)], s.preparing[keyOf(sl)]
		s.mu.Unlock()
		if (prep != "" && prep == profileID) || (cur.State == Launching && cur.Profile != "" && cur.Profile == profileID) {
			return true
		}
		if dirErr != nil {
			continue
		}
		if procs, err := s.procsFor(sl, dir, profileID); err == nil && len(procs) > 0 {
			return true
		}
	}
	return false
}

// find returns the profile of the install the loader is running, and when that process began.
func (s *Service) find(g game.Game) (string, time.Time) {
	all, err := s.profiles.List(g.ID())
	if err != nil {
		return "", time.Time{}
	}
	for _, p := range all {
		if p.Error != "" {
			continue
		}
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

// gameProcs are the running processes of g's install.
func (s *Service) gameProcs(g game.Game) ([]launch.Process, error) {
	ps, err := launch.Processes(s.procDir, game.ProcessNames(g)...)
	if err != nil {
		return nil, err
	}
	var out []launch.Process
	for _, p := range ps {
		if procVisible == nil || s.procDir != procDirRun || procVisible(p) {
			out = append(out, p)
		}
	}
	return s.ownedBy(g, out), nil
}

func (s *Service) seen(g game.Game) func() bool {
	return func() bool {
		ps, err := s.gameProcs(g)
		return err == nil && len(ps) > 0
	}
}

// poll syncs the stored state with the processes, and reports whether the game is still worth watching.
func (s *Service) poll(g game.Game) bool {
	procs, _ := s.gameProcs(g)
	alive := len(procs) > 0
	var profileID string
	var began time.Time
	// A loader's processes are among the game's, so with none running no profile is running either, and the
	// per-profile search can be skipped.
	if alive {
		profileID, began = s.find(g)
	}
	cur := s.current(g)
	switch {
	case cur.State == Launching:
	case cur.State == Idle && profileID != "":
		// A game Mortar did not start has its own log; the last session's buffer is stale.
		s.mu.Lock()
		delete(s.logs, keyOf(g))
		s.mu.Unlock()
		s.set(Status{Game: g.ID(), Install: installOf(g), State: Running, Profile: profileID, Since: sinceOr(began)})
	case cur.State == Idle && alive:
		s.mu.Lock()
		delete(s.logs, keyOf(g))
		s.mu.Unlock()
		s.set(Status{Game: g.ID(), Install: installOf(g), State: Running, Since: sinceOr(procs[0].Start)})
	case cur.State == Running && !alive:
		// With no game process left the run is over, whatever the exit waiter is still waiting on.
		log.Printf("launch: %s %s: poll found no game process of install %q (%s)", g.ID(), cur.Profile, installOf(g), s.describeProcs(g))
		s.awaitReap(g)
		s.closed(g, cur, false)
	case cur.State == Running && cur.Profile != "" && profileID == "":
		if s.reapArmed(g) {
			break
		}
		log.Printf("launch: %s %s: poll found the game running but not this profile (%s)", g.ID(), cur.Profile, s.describeProcs(g))
		s.closed(g, cur, false)
	}
	return s.current(g).State != Idle
}

// describeProcs lists the processes named like the game's and the install each belongs to, for the log line that says
// why a run was closed: on Windows the command line is unreadable, so the install is the only thing tying a process
// to the run.
func (s *Service) describeProcs(g game.Game) string {
	ps, err := launch.Processes(s.procDir, game.ProcessNames(g)...)
	if err != nil {
		return "listing processes: " + err.Error()
	}
	if len(ps) == 0 {
		return "no process named " + strings.Join(game.ProcessNames(g), ", ")
	}
	parts := make([]string, 0, len(ps))
	for _, p := range ps {
		id, ok := s.owner(g, p)
		parts = append(parts, fmt.Sprintf("pid %d exe %q install %q owned %v", p.PID, p.Exe, id, ok))
	}
	return strings.Join(parts, "; ")
}

// watch polls every 2 s until the game is idle.
func (s *Service) watch(g game.Game) {
	s.mu.Lock()
	if s.watching[keyOf(g)] {
		s.mu.Unlock()
		return
	}
	s.watching[keyOf(g)] = true
	s.mu.Unlock()
	go func() {
		tick := time.NewTicker(pollEvery)
		defer tick.Stop()
		for range tick.C {
			if !s.poll(g) {
				s.mu.Lock()
				// A launch that began after this poll found the game already marked as watched; keep watching it.
				st, ok := s.status[keyOf(g)]
				if ok && st.State.Active() {
					s.mu.Unlock()
					continue
				}
				s.watching[keyOf(g)] = false
				s.mu.Unlock()
				return
			}
		}
	}()
}

// Busy reports whether any install of the game is launching or running.
func (s *Service) Busy(gameID string) bool {
	st, err := s.Status(gameID)
	return err == nil && st.State.Active()
}

// BusyInstall reports whether the install with this id is launching or running.
func (s *Service) BusyInstall(installID string) bool {
	sl, ok := s.findInstall(installID)
	return ok && s.statusOf(sl).State.Active()
}

// findInstall is the slot of the install with this id.
func (s *Service) findInstall(installID string) (slot, bool) {
	if installID == "" {
		return slot{}, false
	}
	for _, id := range game.Implemented() {
		g := game.Find(id)
		for _, sl := range s.slots(g) {
			if sl.inst == installID {
				return sl, true
			}
		}
	}
	return slot{}, false
}

// AnyBusy reports whether any implemented game is launching or running.
func (s *Service) AnyBusy() bool {
	return slices.ContainsFunc(game.Implemented(), s.Busy)
}

// statusOf is the install's launch state after looking for a game Mortar did not start.
// startFailure is why the last start of the slot failed, "" when it did not.
func (s *Service) startFailure(sl slot) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startFailed[keyOf(sl)]
}

// noteFailure keeps why the game's launch failed, for the run record finishFailed writes and for LaunchFailure.
func (s *Service) noteFailure(g game.Game, why string) {
	s.mu.Lock()
	s.lastFailure[keyOf(g)] = why
	s.mu.Unlock()
}

// LaunchFailure is why the last launch of the game's install failed, "" when it did not; installID "" is the game's
// selected install.
func (s *Service) LaunchFailure(gameID, installID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastFailure[slotKey(gameID, installID)]
}

func (s *Service) statusOf(sl slot) Status {
	if s.poll(sl) {
		s.watch(sl)
	}
	return s.current(sl)
}

// StatusInstall is the launch state of the install with this id.
func (s *Service) StatusInstall(gameID, installID string) (Status, error) {
	sl, err := s.installSlot(gameID, installID)
	if err != nil {
		return Status{}, err
	}
	return s.statusOf(sl), nil
}

// installSlot is the slot of the install with this id, which must belong to the game.
func (s *Service) installSlot(gameID, installID string) (slot, error) {
	sl, ok := s.findInstall(installID)
	if !ok || sl.ID() != gameID {
		return slot{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("install %q of %s is no longer found", installID, gameID))
	}
	return sl, nil
}

// Status returns the launch state of the game's install that is launching or running, else of the selected install.
func (s *Service) Status(gameID string) (Status, error) {
	g, err := game.Require(gameID)
	if err != nil {
		return Status{}, err
	}
	var out Status
	found := false
	for i, sl := range s.slots(g) {
		st := s.statusOf(sl)
		if i == 0 || (!found && st.State.Active()) {
			out = st
		}
		found = found || st.State.Active()
	}
	return out, nil
}

// Start launches the profile. Its outcome arrives as StateEvents: Launching, then Running or Failed, or NoSteam
// when the user must first agree to launch without Steam (direct). A missing or broken loader is installed first.
func (s *Service) Start(ctx context.Context, gameID, profileID string, direct bool) error {
	return s.StartPreset(ctx, gameID, profileID, "", "", direct)
}

// StartPreset is Start with the named launch preset of the profile for this launch only; an empty name is the
// profile's default preset. A non-empty installID launches on that install instead of the profile's pinned one.
func (s *Service) StartPreset(ctx context.Context, gameID, profileID, installID, preset string, direct bool) error {
	// The game outlives the call that started it, so the caller's cancellation does not reach the launch.
	return s.start(context.WithoutCancel(ctx), gameID, profileID, installID, preset, direct)
}

// pinOf is the install a launch of the profile runs on: the one asked for, else the profile's pinned one.
func (s *Service) pinOf(gameID, profileID, installID string) string {
	if installID != "" {
		return installID
	}
	return s.profiles.InstallOf(gameID, profileID)
}

func (s *Service) start(parent context.Context, gameID, profileID, installID, preset string, direct bool) error {
	g, err := game.Require(gameID)
	if err != nil {
		return err
	}
	sl := s.profileSlot(g, profileID)
	if installID != "" {
		if sl, err = s.installSlot(gameID, installID); err != nil {
			return err
		}
	}
	key := keyOf(sl)
	// preparing is claimed under the same lock as the check, so two Starts of one install cannot both pass it.
	s.mu.Lock()
	cur := s.status[key]
	_, busy := s.preparing[key]
	busy = busy || cur.State.Active()
	if !busy {
		s.preparing[key] = profileID
	}
	s.mu.Unlock()
	if busy {
		return usererr.Wrap(usererr.Busy, fmt.Errorf("%s is already running", g.Name()))
	}
	dir, modsDir, err := s.target(g, profileID, installID)
	if err == nil {
		_, err = s.profiles.LaunchSpec(gameID, profileID, preset)
	}
	if err != nil {
		s.donePreparing(sl)
		return err
	}
	// Launching from the moment the start is accepted, so a caller polling the state never sees the idle gap.
	s.set(Status{Game: gameID, Install: sl.inst, State: Launching, Profile: profileID})
	// Even an installed loader goes through EnsureLoader: it waits out an update in progress, which would
	// otherwise launch the game on half-replaced files.
	go func() {
		defer s.donePreparing(sl)
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
		err := s.EnsureLoader(ctx, gameID, s.profileLoader(gameID, profileID), true)
		if err != nil {
			err = fmt.Errorf("could not install %s: %w", game.LoaderName(g.ID(), s.profileLoader(gameID, profileID)), err)
		} else {
			// Reading the profile takes its lock, so a change to its mods already under way finishes first; any
			// later one sees the profile as running.
			if err = s.profiles.Rebuild(gameID, profileID); err == nil {
				// The run gets parent, not ctx: ctx is cancelled as soon as this goroutine returns, which would
				// end the game Mortar just started.
				err = s.begin(parent, sl, launchTarget{profileID: profileID, install: installID, preset: preset, dir: dir, modsDir: modsDir}, direct, false)
			}
		}
		if err != nil {
			s.failBeforeRun(sl, profileID, err)
		}
	}()
	return nil
}

// failBeforeRun ends a launch that failed before the game was started. The cause goes to a fresh console and a failed
// run record, so the previous run's session is never read as this one's, and to a caller waiting on the launch.
func (s *Service) failBeforeRun(sl slot, profileID string, err error) {
	started := time.Now()
	_, why := usererr.Parse(err.Error())
	s.mu.Lock()
	s.logs[keyOf(sl)] = session{buf: &launch.Buffer{}, profile: profileID, started: started}
	s.mu.Unlock()
	s.say(sl, profileID, why)
	s.noteFailure(sl, why)
	s.record(sl, profileID, started, true)
	s.failStart(Status{Game: sl.ID(), Install: sl.inst, State: Failed, Profile: profileID, Error: err.Error()})
}

// ForcesLoader reports whether Steam's launch options will start the loader even for a vanilla launch.
func (s *Service) ForcesLoader(gameID string) (bool, error) {
	g, err := game.Require(gameID)
	if err != nil {
		return false, err
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
	return game.StartsLoader(gameID, opts)
}

// StartVanilla launches the game without a profile mods folder. On Windows that is steam -applaunch
// with no extra arguments (Steam launch options may still force SMAPI). On Linux SMAPI replaced the
// game launcher, so Mortar starts StardewValley-original directly.
func (s *Service) StartVanilla(parent context.Context, gameID string, direct bool) error {
	g, err := game.Require(gameID)
	if err != nil {
		return err
	}
	sl := s.selectedSlot(g)
	key := keyOf(sl)
	s.mu.Lock()
	cur := s.status[key]
	_, busy := s.preparing[key]
	busy = busy || cur.State.Active()
	if !busy {
		s.preparing[key] = ""
	}
	s.mu.Unlock()
	if busy {
		return usererr.Wrap(usererr.Busy, fmt.Errorf("%s is already running", g.Name()))
	}
	dir, err := game.InstallDir(s.home, s.settings.Get(), g.ID())
	if err != nil {
		s.donePreparing(sl)
		return err
	}
	if dir == "" {
		s.donePreparing(sl)
		return fmt.Errorf("%s is not installed", g.Name())
	}
	go func() {
		defer s.donePreparing(sl)
		if err := s.begin(parent, sl, launchTarget{dir: dir}, direct, true); err != nil {
			s.set(Status{Game: gameID, Install: sl.inst, State: Failed, Error: plainLaunchError(err, dir)})
		}
	}()
	return nil
}

// target returns the game's install folder and the profile's mods folder.
func (s *Service) target(g game.Game, profileID, installID string) (dir, modsDir string, err error) {
	inst, err := game.ResolveInstall(s.home, s.settings.Get(), g.ID(), s.pinOf(g.ID(), profileID, installID))
	if err != nil {
		return "", "", err
	}
	if inst.Dir == "" {
		return "", "", fmt.Errorf("%s is not installed", g.Name())
	}
	dir = inst.Dir
	if err := s.profiles.Rebuild(g.ID(), profileID); err != nil {
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
	g, err := game.Require(gameID)
	if err != nil {
		return fail(err)
	}
	if _, _, err := s.target(g, profileID, ""); err != nil {
		return fail(err)
	}
	inst, err := game.ResolveInstall(s.home, s.settings.Get(), gameID, s.profiles.InstallOf(gameID, profileID))
	if err != nil {
		return fail(err)
	}
	plan, err := s.launchPlan(context.Background(), g, inst, profileID, launchplan.ModeProfile, options, prefix, env)
	if err != nil {
		return fail(err)
	}
	cmd, err := s.Starter.Command(runtime.GOOS, inst, plan)
	if err != nil {
		return fail(err)
	}
	preview.Env = cmd.Env
	preview.Argv = append([]string{cmd.Name}, cmd.Args...)
	return preview
}

func (s *Service) donePreparing(g game.Game) {
	s.mu.Lock()
	delete(s.preparing, keyOf(g))
	st := s.status[keyOf(g)]
	s.mu.Unlock()
	if !st.State.Active() && s.Unlocked != nil {
		s.Unlocked()
	}
}

// launchTarget is what a launch runs: the profile and preset, and the game and mods folders.
type launchTarget struct {
	profileID, install, preset, dir, modsDir string
}

// begin starts the launch once the loader is in place.
func (s *Service) begin(ctx context.Context, g game.Game, t launchTarget, direct, vanilla bool) error {
	gameID := g.ID()
	profileID, dir, modsDir := t.profileID, t.dir, t.modsDir
	var spec profile.LaunchSpec
	if !vanilla && profileID != "" {
		var err error
		if spec, err = s.profiles.LaunchSpec(gameID, profileID, t.preset); err != nil {
			return err
		}
	}
	if ps, err := s.gameProcs(g); err == nil && len(ps) > 0 {
		return usererr.Wrap(usererr.Busy, fmt.Errorf("%s is already running", g.Name()))
	}
	ov := spec.Overrides(launchOverrides(s.profiles, g.ID(), profileID))
	showConsole := settings.ResolveAt(s.settings.Get(), "showSmapiConsole", settings.Scope{Game: g.ID(), Install: installOf(g), Profile: profileID}, ov) == "true"
	mode := launchplan.ModeProfile
	if vanilla {
		mode = launchplan.ModeVanilla
	}
	env := game.StartEnv{Direct: direct, HideWindow: !showConsole, Ready: s.seen(g)}
	var measure, sample bool
	var startupBefore map[string]bool
	if waitOnChild(direct, vanilla) {
		env.OnExit = func(x launch.Exit) { s.finishWait(g, x) }
		s.armReap(g)
	}
	pin := ""
	if !vanilla {
		pin = s.pinOf(g.ID(), profileID, t.install)
	}
	inst, err := game.ResolveInstall(s.home, s.settings.Get(), g.ID(), pin)
	if err != nil {
		return err
	}
	var plan *launchplan.Plan
	if !vanilla && profileID != "" {
		st := s.settings.Get()
		l, _ := s.loaderOf(g.ID(), profileID)
		measure, err = prepareStartup(l, g.ID(), modsDir)
		if err != nil {
			return err
		}
		// The sampler reads .NET's diagnostics port, which SMAPI's runtime has and Unity's Mono does not.
		sample = measure && l.ID() == smapi.ID
		if sample {
			profileDir, profileErr := s.profiles.ProfileDir(g.ID(), profileID)
			if profileErr != nil {
				log.Printf("startup sampler: %s: %v", g.ID(), profileErr)
				measure, sample = false, false
			} else if startupBefore, profileErr = startupReportIDs(filepath.Join(profileDir, startupDir)); profileErr != nil {
				log.Printf("startup sampler: %s: %v", g.ID(), profileErr)
				measure, sample = false, false
			}
		}
		cfg := overlay.BridgeConfig{OverlayEnabled: st.OverlayEnabled, OverlayPort: st.OverlayPort, OverlayToken: st.OverlayToken, StartupProfile: measure}
		if err := overlay.ApplyToMods(modsDir, cfg); err != nil {
			return err
		}
		if armer, ok := l.(loader.OverlayArmer); ok {
			dir, dirErr := s.profiles.ProfileDir(g.ID(), profileID)
			if dirErr != nil {
				return dirErr
			}
			if armErr := armer.ArmOverlay(dir, st.OverlayEnabled, st.OverlayPort, st.OverlayToken); armErr != nil {
				return armErr
			}
		}
	}
	if plan, err = s.planProfile(ctx, g, inst, profileID, mode, spec.Options, spec.Prefix, spec.Env); err != nil {
		return err
	}
	env.LogFile, _, _ = s.logPath(g.ID(), profileID)
	// The install is locked from the preparing claim until the waiter has unwound: a second launch finds the game busy.
	dep, err := s.deployProfile(ctx, g.ID(), inst, profileID, plan)
	if err != nil {
		return err
	}
	store := inst.Store
	if store == game.StoreGOG || store == game.StoreGOGHeroic || store == game.StoreMinigalaxy || store == game.StoreLutris || store == game.StoreBottles || store == game.StoreEA {
		env.Direct = true
	}
	if st, status := steam.Locate(s.home); status == steam.Found {
		env.Steam = &st
	}
	if store == game.StoreFlatpakSteam {
		for _, one := range steam.LocateAll(s.home) {
			if one.Kind == steam.KindFlatpak {
				env.Steam = &one
				break
			}
		}
	}
	var restore *settingsRestore
	var settingsMissing bool
	var backupErr error
	if !vanilla && profileID != "" {
		restore, settingsMissing, err = s.prepareGameSettings(g.ID(), profileID, t.install)
		if err != nil {
			dep.unwind(ctx)
			return err
		}
		backupErr = s.backupChangedSaves(g.ID(), profileID, t.install, dir)
		err = s.swapSaves(ctx, g.ID(), inst, profileID, t.install, dep)
		if err == nil {
			err = s.swapOptions(ctx, g.ID(), inst, profileID, t.install, dep)
		}
		if err != nil {
			dep.unwind(ctx)
			if restoreErr := s.restoreGameSettings(restore); restoreErr != nil {
				s.reportSettingsRestore(g, profileID, restoreErr)
			}
			return err
		}
		s.clearCaches(g.ID(), profileID, t.install)
	}
	if err := s.ensureRuntime(inst, plan.RuntimeReqs, env.Direct || plan.Exe != ""); err != nil {
		if errors.Is(err, errNoPrefix) {
			err = usererr.New(usererr.Invalid, "Start "+g.Name()+" once from Steam and quit it, so Proton can create its prefix; then Play from Mortar. Or set the game's Default launch to Direct.")
		}
		dep.unwind(ctx)
		if restoreErr := s.restoreGameSettings(restore); restoreErr != nil {
			s.reportSettingsRestore(g, profileID, restoreErr)
		}
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	logCap, _ := strconv.Atoi(settings.ResolveAt(s.settings.Get(), "consoleLogCap", settings.Scope{Game: gameID, Profile: profileID}, launchOverrides(s.profiles, gameID, profileID)))
	buf := &launch.Buffer{Cap: logCap}
	started := time.Now()
	mods := s.profileModRefs(gameID, profileID)
	s.mu.Lock()
	s.logs[keyOf(g)], s.stop[keyOf(g)] = session{
		buf:             buf,
		profile:         profileID,
		preset:          spec.Name,
		vanilla:         vanilla,
		started:         started,
		mods:            mods,
		restore:         restore,
		deployed:        dep,
		settingsMissing: settingsMissing,
	}, cancel
	s.mu.Unlock()
	s.set(Status{Game: gameID, State: Launching, Profile: profileID, Install: installOf(g), Since: started.UnixMilli()})
	if settingsMissing {
		s.say(g, profileID, "startup_preferences is missing; skipped profile game settings.")
	}
	if backupErr != nil {
		msg := fmt.Sprintf("Could not back up saves before playing: %v.", backupErr)
		s.say(g, profileID, msg)
		s.emit(BackupWarningEvent, BackupWarning{Game: gameID, Profile: profileID, Error: backupErr.Error()})
	}
	s.watch(g)
	go s.followRunning(runCtx, g)
	if sample {
		stopped := make(chan struct{})
		s.mu.Lock()
		s.sampled[keyOf(g)] = stopped
		s.mu.Unlock()
		go s.sampleStartup(runCtx, g, profileID, modsDir, startupBefore, sync.OnceFunc(func() { close(stopped) }))
	}
	go s.run(runCtx, g, profileID, launchRun{inst: inst, plan: plan, env: env, vanilla: vanilla}, buf)
	return nil
}

// waitSampled waits for a measured launch's sampler to stop its session, so stopping the game does not cut off the
// method rundown the samples are resolved with.
func (s *Service) waitSampled(g game.Game, limit time.Duration) {
	s.mu.Lock()
	stopped := s.sampled[keyOf(g)]
	delete(s.sampled, keyOf(g))
	s.mu.Unlock()
	if stopped == nil {
		return
	}
	select {
	case <-stopped:
	case <-time.After(limit):
	}
}

// playedSavesDir is the folder holding the saves this launch plays: the profile's own when it keeps its saves
// separate, which the launch swaps in after the backup, else the game's.
func (s *Service) playedSavesDir(set settings.Settings, gameID, profileID, installID string) (string, error) {
	if s.profiles.SeparateSaves(gameID, profileID) {
		return s.profiles.SavesFolder(gameID, profileID)
	}
	return game.SavesDir(s.home, set, gameID, s.pinOf(gameID, profileID, installID))
}

func (s *Service) backupChangedSaves(gameID, profileID, installID, installDir string) error {
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
	set := s.settings.Get()
	recorded := set.LastPlayed[gameID].GameVersion
	installedStatus, _ := game.LoaderStatus(gameID, s.profileLoader(gameID, profileID), installDir, set.Loaders)
	installed := installedStatus.GameVersion
	ov := launchOverrides(s.profiles, gameID, profileID)
	mode := settings.ResolveAt(set, "backupBeforePlay", settings.Scope{Game: gameID, Install: s.pinOf(gameID, profileID, installID), Profile: profileID}, ov)
	if !backupNeeded(mode, events, lastRun, recorded, installed) {
		return nil
	}
	savesDir, err := s.playedSavesDir(set, gameID, profileID, installID)
	if err != nil {
		return err
	}
	dataDir, err := s.dataDir()
	if err != nil {
		return err
	}
	target, err := backup.TargetFor(dataDir, set, gameID, ov)
	if err != nil {
		return err
	}
	_, err = backup.Saves(
		game.SaveLayout(gameID, savesDir),
		target.Dir,
		target.Keep,
		time.Now(),
		backup.Cause{Profile: profileID, Kind: backup.KindLaunch},
	)
	return err
}

func launchOverrides(profiles *profile.Store, gameID, profileID string) map[string]string {
	if profiles == nil || profileID == "" {
		return nil
	}
	all, err := profiles.List(gameID)
	if err != nil {
		return nil
	}
	for _, p := range all {
		if p.ID == profileID {
			return p.PrefOverrides()
		}
	}
	return nil
}

func backupNeeded(mode string, events []profile.HistoryEvent, lastRun time.Time, recorded, installed string) bool {
	return settings.ShouldBackupBeforePlay(mode, changedSinceLastRun(events, lastRun), loader.VersionChanged(recorded, installed))
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
func (s *Service) collect(g game.Game, profileID string, buf *launch.Buffer) func([]string) {
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
		current := s.logs[keyOf(g)].buf == buf
		s.mu.Unlock()
		if !current {
			return
		}
		for _, e := range entries {
			buf.Add(e)
		}
		s.emit(LineEvent, Lines{Game: g.ID(), Profile: profileID, Entries: entries})
	}
}

// Lines returns the profile's log, oldest first, at most launch.MaxLines: this session's when Mortar launched the
// game, else the log file on disk, which is what the last session left after a crash. It is empty when the log is
// another profile's.
func (s *Service) Lines(gameID, profileID string) ([]launch.Entry, error) {
	g, err := game.Require(gameID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	sess, ok := s.logs[keyOf(s.profileSlot(g, profileID))]
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
	path, own, err := s.logPath(g.ID(), profileID)
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
	if !own && !launch.LogOwnedBy(text, s.home, modsDir) {
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

// launchRun is a launch ready to start: the install it runs, the plan loaders contributed to and how the store starts it.
type launchRun struct {
	inst    game.Install
	plan    *launchplan.Plan
	env     game.StartEnv
	vanilla bool
}

func (s *Service) run(ctx context.Context, g game.Game, profileID string, r launchRun, buf *launch.Buffer) {
	err := s.Starter.Start(ctx, r.inst, r.plan, r.env, s.collect(g, profileID, buf))
	if err != nil {
		s.mu.Lock()
		sess := s.logs[keyOf(g)]
		s.mu.Unlock()
		if sess.buf == buf && sess.profile == profileID {
			sess.deployed.unwind(context.WithoutCancel(ctx))
			if restoreErr := s.restoreGameSettings(sess.restore); restoreErr != nil {
				s.reportSettingsRestore(g, profileID, restoreErr)
			}
		}
	}
	var f *launch.Failure
	var exited *launch.ExitError
	switch {
	case err == nil:
		s.set(Status{Game: g.ID(), Install: installOf(g), State: Running, Profile: profileID, Since: time.Now().UnixMilli()})
		if r.vanilla {
			s.say(g, profileID, "Started without mods")
		}
		if !waitOnChild(r.env.Direct, r.vanilla) {
			s.armReap(g)
			go s.awaitPID(g)
		}
	case errors.Is(err, launch.ErrNoSteam):
		s.clearReap(g)
		s.set(Status{Game: g.ID(), Install: installOf(g), State: NoSteam, Profile: profileID})
	case errors.As(err, &exited):
		s.clearReap(g)
		if len(buf.Lines()) == 0 {
			s.say(g, profileID, fmt.Sprintf("%s exited with code %d.", game.LoaderName(g.ID(), s.profileLoader(g.ID(), profileID)), exited.Code))
			for _, line := range exited.Output {
				s.say(g, profileID, line)
			}
		}
		s.noteFailure(g, plainLaunchError(err, r.inst.Dir))
		s.noteStartedProcessExit(g, profileID, buf)
		s.finishFailed(g, profileID, buf)
		s.set(Status{Game: g.ID(), Install: installOf(g), State: Failed, Profile: profileID, Error: plainLaunchError(err, r.inst.Dir), Cause: causeFromBuffer(s, g, profileID, buf)})
	case errors.As(err, &f):
		s.clearReap(g)
		if f.Hint == launch.HintSteamClient || f.Hint == launch.HintWine || f.Hint == launch.HintMissingExe {
			s.say(g, profileID, f.Error())
		}
		s.noteFailure(g, f.Error())
		s.finishFailed(g, profileID, buf)
		s.failStart(Status{Game: g.ID(), Install: installOf(g), State: Failed, Profile: profileID, Hint: f.Hint, Error: f.Error(), Cause: causeFromBuffer(s, g, profileID, buf)})
	default:
		s.clearReap(g)
		s.noteFailure(g, plainLaunchError(err, r.inst.Dir))
		s.finishFailed(g, profileID, buf)
		s.failStart(Status{Game: g.ID(), Install: installOf(g), State: Failed, Profile: profileID, Error: plainLaunchError(err, r.inst.Dir), Cause: causeFromBuffer(s, g, profileID, buf)})
	}
}

func causeFromBuffer(s *Service, g game.Game, profileID string, buf *launch.Buffer) *Cause {
	cause := s.cause(g.ID(), profileID, launch.FormatLog(buf.Lines()))
	if cause.ModName == "" {
		return nil
	}
	return &cause
}

func plainLaunchError(err error, dir string) string {
	if errors.Is(err, context.Canceled) {
		return "The launch was interrupted. Try again."
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrPermission) {
		return fmt.Sprintf("Mortar could not start the game files at %s.", dir)
	}
	return err.Error()
}

// Stop terminates the loader process of the profile the game is running, in whichever install it runs.
func (s *Service) Stop(ctx context.Context, gameID string) error {
	g, err := game.Require(gameID)
	if err != nil {
		return err
	}
	for _, sl := range s.slots(g) {
		if s.stoppable(sl) {
			return s.stopSlot(ctx, sl)
		}
	}
	return fmt.Errorf("%s is not running", g.Name())
}

// StopInstall stops the game running from the install with this id.
func (s *Service) StopInstall(ctx context.Context, installID string) error {
	sl, ok := s.findInstall(installID)
	if !ok || !s.stoppable(sl) {
		return usererr.Wrap(usererr.NotFound, fmt.Errorf("install %q is not running", installID))
	}
	return s.stopSlot(ctx, sl)
}

// stoppable reports whether the slot runs, by its stored state first: polling would close a run whose game is
// already gone, and stopping that run is what closes it.
func (s *Service) stoppable(sl slot) bool {
	return s.current(sl).State == Running || s.statusOf(sl).State == Running
}

func (s *Service) stopSlot(ctx context.Context, g slot) error {
	cur := s.current(g)
	// One game runs per install, so its processes are the run's; a profile search misses a loader inside the game.
	procs, err := s.gameProcs(g)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.stopping[keyOf(g)] = true
	if sess, ok := s.logs[keyOf(g)]; ok {
		sess.stoppedAt = time.Now()
		s.logs[keyOf(g)] = sess
	}
	s.mu.Unlock()
	var errs []error
	for _, p := range procs {
		errs = append(errs, launch.Stop(ctx, p, stopGrace))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	// A game that had already exited closed on its own.
	s.closed(g, cur, len(procs) > 0)
	return nil
}

// closed ends the game's session with a console line of Mortar's own, so the log does not just stop, then marks
// the game idle. It does nothing when a concurrent poll or Stop already closed the session.
func (s *Service) closed(g game.Game, cur Status, stopped bool) {
	// The exit watcher, the status poll and Stop can all see the same exit; only the first one closes the run.
	key := keyOf(g)
	s.mu.Lock()
	if st, ok := s.status[key]; !ok || st.State != Running || s.closing[key] {
		s.mu.Unlock()
		return
	}
	if s.closing == nil {
		s.closing = map[string]bool{}
	}
	s.closing[key] = true
	s.mu.Unlock()
	if stopped {
		s.mu.Lock()
		if sess, ok := s.logs[keyOf(g)]; ok && !sess.haveExit {
			sess.exit = launch.Exit{Stopped: true}
			sess.haveExit = true
			s.logs[keyOf(g)] = sess
		}
		s.mu.Unlock()
	}
	msg := g.Name() + " closed"
	if stopped {
		msg = g.Name() + " was stopped from Mortar"
	} else {
		s.mu.Lock()
		sessExit := s.logs[keyOf(g)]
		s.mu.Unlock()
		if sessExit.haveExit && launch.ExitCrashed(sessExit.exit) {
			msg = g.Name() + " crashed; " + launch.DescribeExit(sessExit.exit)
		}
	}
	if cur.Since > 0 {
		msg += " after " + time.Since(time.UnixMilli(cur.Since)).Round(time.Second).String()
	}
	s.say(g, cur.Profile, msg+".")
	log.Printf("launch: %s %s: %s", g.ID(), cur.Profile, msg)
	s.mu.Lock()
	sess, ok := s.logs[keyOf(g)]
	s.mu.Unlock()
	if ok && !sess.vanilla && cur.Profile != "" {
		if sess.deployed != nil && sess.deployed.finish != nil {
			sess.deployed.finish()
		}
		if restoreErr := s.restoreGameSettings(sess.restore); restoreErr != nil {
			s.reportSettingsRestore(g, cur.Profile, restoreErr)
		}
		started := sess.started
		if cur.Since > 0 {
			started = time.UnixMilli(cur.Since)
		}
		s.record(g, cur.Profile, started, false, sess.mods)
	}
	s.set(Status{Game: g.ID(), Install: installOf(g), State: Idle})
	s.clearReap(g)
}

func (s *Service) finishFailed(g game.Game, profileID string, buf *launch.Buffer) {
	s.mu.Lock()
	sess, ok := s.logs[keyOf(g)]
	s.mu.Unlock()
	if !ok || sess.vanilla || sess.buf != buf || profileID == "" {
		return
	}
	s.record(g, profileID, sess.started, true, sess.mods)
}

// say adds a console line of Mortar's own to the session of the profile the game runs.
func (s *Service) say(g game.Game, profileID, msg string) {
	e := launch.Entry{Seq: s.seq.Add(1), Time: time.Now().Format(time.TimeOnly), Level: launch.Info, Mod: "Mortar", Message: msg}
	s.mu.Lock()
	sess, ok := s.logs[keyOf(g)]
	s.mu.Unlock()
	if ok && sess.profile == profileID {
		sess.buf.Add(e)
	}
	s.emit(LineEvent, Lines{Game: g.ID(), Profile: profileID, Entries: []launch.Entry{e}})
}

func (s *Service) reportSettingsRestore(g game.Game, profileID string, err error) {
	msg := fmt.Sprintf("Could not restore profile game settings: %v.", err)
	s.say(g, profileID, msg)
	s.emit(SettingsRestoreWarningEvent, SettingsRestoreWarning{Game: g.ID(), Profile: profileID, Error: err.Error()})
}

// Send runs a console command in the running game through the bridge mod in the running profile. The command
// is echoed into the console; its output arrives with the game's own log.
func (s *Service) Send(gameID, command string) error {
	g, err := game.Require(gameID)
	if err != nil {
		return err
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return errors.New("enter a command")
	}
	sl, ok := s.activeSlot(g)
	if !ok || !s.stoppable(sl) {
		return fmt.Errorf("%s is not running", g.Name())
	}
	cur := s.current(sl)
	l, _ := s.loaderOf(gameID, cur.Profile)
	console, ok := l.(loader.Console)
	if !ok {
		return fmt.Errorf("%s has no console", g.Name())
	}
	inst, err := game.ResolveInstall(s.home, s.currentSettings(), gameID, installOf(sl))
	if err != nil {
		return err
	}
	view, err := s.view(g, inst, cur.Profile)
	if err != nil {
		return err
	}
	if view.Companion == "" {
		return fmt.Errorf("%s has no console bridge in this profile", g.Name())
	}
	if _, err := console.Send(context.Background(), loader.Target{Game: gameID, InstallDir: inst.Dir}, view, command); err != nil {
		return err
	}
	s.say(sl, cur.Profile, "> "+command)
	return nil
}

// bridgeFolder is the loader's companion mod's folder in the profile the game runs, under the key that profile holds:
// a Mortar update can bundle a newer companion than the one the running game loaded.
func (s *Service) bridgeFolder(g game.Game, profileID string) (string, error) {
	l, _ := s.loaderOf(g.ID(), profileID)
	companion, ok := l.(loader.WithCompanion)
	if !ok {
		return "", fmt.Errorf("%s has no console bridge", g.Name())
	}
	all, err := s.profiles.List(g.ID())
	if err != nil {
		return "", err
	}
	for _, p := range all {
		if p.Error != "" || p.ID != profileID {
			continue
		}
		for _, e := range p.Entries {
			if e.Source.Kind == profile.SourceMortar {
				modsDir, err := s.profiles.ModsDir(g.ID(), profileID)
				if err != nil {
					return "", err
				}
				return filepath.Join(modsDir, e.Key, companion.Companion().ModFolder), nil
			}
		}
	}
	return "", fmt.Errorf("%s has no console bridge in this profile", g.Name())
}
