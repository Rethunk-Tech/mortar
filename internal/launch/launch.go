// Package launch starts a game, decides whether the launch worked, and finds and stops the running loader process.
package launch

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/go-gameproc"
	"github.com/Rethunk-Tech/go-safefs"

	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
)

// Hint names why a launch failed, for the frontend to explain in the user's language.
type Hint string

const (
	HintNone Hint = ""
	// HintSteam: the game never wrote its log, so Steam may not be running or signed in.
	HintSteam Hint = "steam"
	// HintFlatpakFS: Flatpak Steam cannot read Mortar's data folder until a filesystem override is granted.
	HintFlatpakFS Hint = "flatpak-fs"
	// HintSteamClient: Proton could not load Steam's client library, so the game needs Steam installed and signed in.
	HintSteamClient Hint = "steam-client"
	// HintWine: Wine or Proton stopped on an error before the game came up.
	HintWine Hint = "wine"
	// HintMissingExe: the process to start, or the game's executable, was not found.
	HintMissingExe Hint = "missing-exe"
	// HintEpic: the game never wrote its log, so the Epic Games Launcher may not be running or signed in.
	HintEpic Hint = "epic"
	// HintLauncher: the game never wrote its log, so Heroic or Lutris may not have started it.
	HintLauncher Hint = "launcher"
)

// ErrNoSteam means there is no Steam to launch through; the user may choose to launch directly.
var ErrNoSteam = errors.New("no Steam was found")

// ExitError is a loader process that ended before it rewrote its log.
type ExitError struct {
	Code int
	// Output is the last lines the process wrote, when it wrote any.
	Output []string
}

func (e *ExitError) Error() string {
	if len(e.Output) == 0 {
		return fmt.Sprintf("the loader exited with code %d", e.Code)
	}
	return fmt.Sprintf("the loader exited with code %d: %s", e.Code, e.Output[len(e.Output)-1])
}

// Failure is a launch that did not start the game.
type Failure struct {
	Hint Hint
	Err  error
}

func (f *Failure) Error() string { return f.Err.Error() }
func (f *Failure) Unwrap() error { return f.Err }

// Request is what a game needs to launch one profile.
type Request struct {
	InstallDir string
	// ModsDir is the profile's absolute mods folder.
	ModsDir string
	// ExtraArgs are extra SMAPI arguments from the profile's launch options.
	ExtraArgs []string
	// Prefix and Env apply only to direct launches.
	Prefix []string
	Env    []string
	// Steam is nil when no Steam was found.
	Steam *steam.Steam
	// Direct launches the loader without Steam, after the user agreed to lose the overlay and playtime.
	Direct bool
	// HideWindow hides the process window (Windows SMAPI console) when true.
	HideWindow bool
	// Vanilla starts the game without the loader or a profile mods folder.
	Vanilla bool
	// Seen reports that a vanilla launch succeeded, when the game process is running. Ignored otherwise.
	Seen func() bool
	// OnExit reports how the loader process ended after the game has started. Not used for Steam relays.
	OnExit func(Exit)
}

// Runner starts a command and returns without waiting for it to finish. exited receives the Wait
// error (nil on a zero exit); nil means the runner will not report an exit.
type Runner func(dir, name string, args ...string) (exited <-chan error, err error)

// Start is the Runner that runs the real command, reaping it in the background.
func Start(dir, name string, args ...string) (<-chan error, error) {
	return StartWithEnv(context.Background(), nil, dir, name, args...)
}

// ReleaseOnExit waits for a command whose output nobody reads (a user's tool) and then frees its captured output,
// which Run does for a game.
func ReleaseOnExit(exited <-chan error) {
	if err := <-exited; err != nil {
		slog.Debug("tool exited", "err", err)
	}
	if out := captureOf(exited); out != nil {
		captured.Delete(exited)
		out.release()
	}
}

// StartWithEnv starts a process with additional environment variables; ctx ending kills it, so a game the caller
// does not want tied to its request passes a context without cancellation.
func StartWithEnv(ctx context.Context, env []string, dir, name string, args ...string) (<-chan error, error) {
	return startCmd(ctx, env, dir, name, args, false)
}

func StartHidden(ctx context.Context, env []string, dir, name string, args ...string) (<-chan error, error) {
	return startCmd(ctx, env, dir, name, args, true)
}

// commandEnv is the host's environment plus env for a child starting in dir. exec only fills in PWD when no environment
// is given, and an AppImage's own PWD is scrubbed as a path inside its mount; GE-Proton's protonfixes reads PWD unguarded.
func commandEnv(dir string, env []string) []string {
	out := append(hostenv.Environ(), env...)
	if dir != "" {
		out = append(out, "PWD="+dir)
	}
	return out
}

func startCmd(ctx context.Context, env []string, dir, name string, args []string, hide bool) (<-chan error, error) {
	cmd := flatpak.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = commandEnv(dir, env)
	if runtime.GOOS == "windows" && !hide {
		return gameproc.StartOwnConsole(cmd.Env, dir, name, args)
	}
	gameproc.PrepareTree(cmd, hide)
	// Stdin is a pipe held open until the process exits: SMAPI reads console commands in a loop that spins a whole
	// core on an input already at end of file (/dev/null, NUL), and blocks quietly on an open one.
	stdin, hold, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdin = stdin
	out := newCapture(cmd)
	err = cmd.Start()
	_ = stdin.Close()
	if err != nil {
		_ = hold.Close()
		out.release()
		return nil, err
	}
	pid := cmd.Process.Pid
	if err := gameproc.TrackTree(pid); err != nil && !errors.Is(err, gameproc.ErrNotTracked) {
		gameproc.KillTree(pid)
		_ = cmd.Wait()
		gameproc.UntrackTree(pid)
		_ = hold.Close()
		out.release()
		return nil, err
	}
	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		gameproc.UntrackTree(pid)
		_ = hold.Close()
		out.removeIfReleased()
		done <- err
	}()
	exited := (<-chan error)(done)
	captured.Store(exited, out)
	return exited, nil
}

// Command is one process to start, plus how to tell it worked.
type Command struct {
	Dir  string
	Name string
	Args []string
	Env  []string
	// LogFile is the file the game rewrites when it starts.
	LogFile string
	// Failure names the hint to give when the log never appears.
	Failure Hint
	// Relay marks a process that hands the launch to another and exits 0, such as `steam -applaunch`; a zero exit
	// says nothing about the game, so only the log decides. A non-zero exit is still a failure.
	Relay bool
	// Via is the launcher a Relay hands the launch to, by its launcher id (steam, flatpak-steam, heroic, lutris, epic).
	Via string
	// Ready, when set, is the success rule instead of a rewritten log: a vanilla launch waits for the game process.
	Ready func() bool
	// Alive reports that the game's process is running. A game whose log is slow to appear (a big mod list, a slow
	// disk) has still started when its process is up once the launch is slow; the log is followed once it comes.
	Alive func() bool
	// OnExit reports how the started process ended after the game has started. Ignored for Relay commands.
	OnExit func(Exit)
	// OnStage reports each Stage the launch reaches, once each and in order.
	OnStage func(Stage)
}

// Stage is a step a launch reaches on its way to the game being up.
type Stage string

const (
	// StageHanded is a relay such as `steam -applaunch` having handed the launch on.
	StageHanded Stage = "handed"
	// StageUp is the game's process running.
	StageUp Stage = "up"
	// StageLoader is the loader having started its log.
	StageLoader Stage = "loader"
)

const clockSlack = 50 * time.Millisecond

// SlowAfter is how long a launch runs before it counts as slow: the player is told it is taking a while, and a game
// whose process is up has started even without its log.
const SlowAfter = 90 * time.Second

// Timing controls how long Run waits for the game; the zero value gives up after 5 minutes, polling every 250 ms.
type Timing struct {
	Timeout time.Duration
	Slow    time.Duration
	Poll    time.Duration
}

func (t Timing) withDefaults() Timing {
	t.Timeout = cmp.Or(t.Timeout, 5*time.Minute)
	t.Slow = cmp.Or(t.Slow, SlowAfter)
	t.Poll = cmp.Or(t.Poll, 250*time.Millisecond)
	return t
}

// Run starts c and waits for success: Ready when set, otherwise the log file rewritten after the start, sending
// each batch of new log lines to onLines as it appears. It returns a *Failure when that does not happen within
// the timeout. Once the game has started, a log keeps being followed until ctx is done, with one last read then;
// ctx must outlive the game.
func Run(ctx context.Context, run Runner, c Command, tm Timing, onLines func([]string)) error {
	tm = tm.withDefaults()
	began := time.Now()
	exited, err := run(c.Dir, c.Name, c.Args...)
	if err != nil {
		return &Failure{Hint: c.Failure, Err: err}
	}
	a := &attempt{
		c: c, tm: tm, onLines: onLines, began: began, exited: exited, out: captureOf(exited),
		reached: map[Stage]bool{StageHanded: false, StageUp: false, StageLoader: false},
		// File timestamps come from a coarse kernel clock that can trail time.Now by a few milliseconds.
		tail: tailer{path: c.LogFile, since: began.Add(-clockSlack), onLines: onLines},
	}
	// A game that comes up with an exit callback keeps its capture until it exits, so the callback gets the output.
	if a.out != nil {
		captured.Delete(exited)
		defer func() {
			if !a.keep {
				a.out.release()
			}
		}()
	}
	return a.wait(ctx)
}

// attempt is the state of one Run: the process, its output capture and the stages reached so far.
type attempt struct {
	c       Command
	tm      Timing
	onLines func([]string)
	began   time.Time
	exited  <-chan error
	out     *capture
	tail    tailer
	reached map[Stage]bool
	keep    bool
}

// stalled reports a known cause in the process's output while the game is not up, and ends the process: Wine
// leaves an assertion dialog open forever, so waiting out the timeout would only hide the reason.
func (a *attempt) stalled(exited bool) *Failure {
	hint, line, found := Diagnose(a.out.tail(), exited)
	if !found {
		return nil
	}
	a.out.stop()
	return &Failure{Hint: hint, Err: errors.New(line)}
}

func (a *attempt) ok() bool {
	if a.c.Ready != nil {
		return a.c.Ready()
	}
	return a.tail.poll()
}

func (a *attempt) started(ctx context.Context) {
	if a.c.LogFile != "" {
		go follow(ctx, &a.tail, a.tm.Poll)
	}
	if a.c.OnExit != nil && a.exited != nil && !a.c.Relay {
		ch := a.exited
		a.keep = true
		go func() {
			x := WaitError(<-ch)
			x.Output = a.out.tail()
			a.out.release()
			a.c.OnExit(x)
		}()
	}
}

func (a *attempt) stage(st Stage) {
	if !a.reached[st] && a.c.OnStage != nil {
		a.c.OnStage(st)
	}
	a.reached[st] = true
}

// up reads the process table until the game is first seen; after that it is known to have come up.
func (a *attempt) up() bool {
	if !a.reached[StageUp] && a.c.Alive != nil && a.c.Alive() {
		a.stage(StageUp)
	}
	return a.reached[StageUp]
}

func (a *attempt) succeed(ctx context.Context) error {
	a.stage(StageUp)
	if a.c.LogFile != "" {
		a.stage(StageLoader)
	}
	a.started(ctx)
	return nil
}

// onExit decides what the process ending means; done is false when a relay launcher handed the game over.
func (a *attempt) onExit(ctx context.Context, waitErr error) (done bool, err error) {
	if a.ok() {
		return true, a.succeed(ctx)
	}
	if a.c.Relay && waitErr == nil {
		a.exited = nil
		a.stage(StageHanded)
		return false, nil
	}
	if f := a.stalled(true); f != nil {
		return true, f
	}
	return true, &ExitError{Code: waitCode(waitErr), Output: lastLines(a.out.tail(), 5)}
}

func (a *attempt) wait(ctx context.Context) error {
	deadline := time.NewTimer(a.tm.Timeout)
	defer deadline.Stop()
	tick := time.NewTicker(a.tm.Poll)
	defer tick.Stop()
	for {
		alive := a.up()
		if a.ok() || time.Since(a.began) >= a.tm.Slow && alive {
			return a.succeed(ctx)
		}
		if f := a.stalled(false); f != nil {
			return f
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			if a.up() {
				return a.succeed(ctx)
			}
			return &Failure{Hint: a.c.Failure, Err: errors.New("the game did not start in time")}
		case waitErr := <-a.exited:
			if done, err := a.onExit(ctx, waitErr); done {
				return err
			}
		case <-tick.C:
		}
	}
}

func waitCode(err error) int {
	if err == nil {
		return 0
	}
	var c interface{ ExitCode() int }
	if errors.As(err, &c) {
		return c.ExitCode()
	}
	return 1
}

func follow(ctx context.Context, tail *tailer, every time.Duration) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			tail.poll()
			return
		case <-tick.C:
			tail.poll()
		}
	}
}

// tailer reads a log from the start once its mtime is newer than since, handing over complete lines.
type tailer struct {
	path    string
	since   time.Time
	onLines func([]string)
	offset  int64
	// followed is the file a glob path currently names.
	followed string
	// rest is a trailing partial line, held until its newline arrives.
	rest string
}

// poll reports whether the log has been rewritten since the launch, emitting any new lines first.
func (t *tailer) poll() bool {
	if t.path == "" {
		return false
	}
	if strings.ContainsAny(t.path, "*?[") {
		// A loader that starts a log file of its own name each run: follow the newest one written since the launch.
		path := NewestMatch(t.path, t.since)
		if path == "" {
			return false
		}
		if path != t.followed {
			t.followed, t.offset, t.rest = path, 0, ""
		}
		return t.pollFile(path)
	}
	return t.pollFile(t.path)
}

func (t *tailer) pollFile(path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.ModTime().After(t.since) {
		return false
	}
	f, err := safefs.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	if st.Size() < t.offset {
		t.offset, t.rest = 0, ""
	}
	if _, err := f.Seek(t.offset, io.SeekStart); err != nil {
		return false
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return false
	}
	t.offset += int64(len(data))
	lines := strings.Split(t.rest+string(bytes.ToValidUTF8(data, nil)), "\n")
	t.rest = lines[len(lines)-1]
	var batch []string
	for _, l := range lines[:len(lines)-1] {
		if l = strings.TrimRight(l, "\r"); l != "" {
			batch = append(batch, l)
		}
	}
	if len(batch) > 0 && t.onLines != nil {
		t.onLines(batch)
	}
	return true
}

// NewestMatch is the file matching the glob pattern that was written last, "" when none was written after since (the
// zero time takes any).
func NewestMatch(pattern string, since time.Time) string {
	names, _ := filepath.Glob(pattern)
	best, bestTime := "", since
	for _, n := range names {
		if st, err := os.Stat(n); err == nil && st.Mode().IsRegular() && (best == "" && since.IsZero() || st.ModTime().After(bestTime)) {
			best, bestTime = n, st.ModTime()
		}
	}
	return best
}
