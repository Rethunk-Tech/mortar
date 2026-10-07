// Package launch starts a game, decides whether the launch worked, and finds and stops the running loader process.
package launch

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// Hint names why a launch failed, for the frontend to explain in the user's language.
type Hint string

const (
	HintNone Hint = ""
	// HintSteam: the game never wrote its log, so Steam may not be running or signed in.
	HintSteam Hint = "steam"
	// HintLaunchOptions: Steam's launch options for the game lack the loader's line.
	HintLaunchOptions Hint = "launch-options"
	// HintFlatpakFS: Flatpak Steam cannot read Mortar's data folder until a filesystem override is granted.
	HintFlatpakFS Hint = "flatpak-fs"
	// HintSteamClient: Proton could not load Steam's client library, so the game needs Steam installed and signed in.
	HintSteamClient Hint = "steam-client"
	// HintWine: Wine or Proton stopped on an error before the game came up.
	HintWine Hint = "wine"
	// HintMissingExe: the process to start, or the game's executable, was not found.
	HintMissingExe Hint = "missing-exe"
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

// StartWithEnv starts a process with additional environment variables; ctx ending kills it, so a game the caller
// does not want tied to its request passes a context without cancellation.
func StartWithEnv(ctx context.Context, env []string, dir, name string, args ...string) (<-chan error, error) {
	return startCmd(ctx, env, dir, name, args, false)
}

func StartHidden(ctx context.Context, env []string, dir, name string, args ...string) (<-chan error, error) {
	return startCmd(ctx, env, dir, name, args, true)
}

func startCmd(ctx context.Context, env []string, dir, name string, args []string, hide bool) (<-chan error, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	hideWindow(cmd, hide)
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
	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		_ = hold.Close()
		done <- err
	}()
	exited := (<-chan error)(done)
	if out != nil {
		captured.Store(exited, out)
	}
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
	// Ready, when set, is the success rule instead of a rewritten log: a vanilla launch waits for the game process.
	Ready func() bool
	// OnExit reports how the started process ended after the game has started. Ignored for Relay commands.
	OnExit func(Exit)
}

const clockSlack = 50 * time.Millisecond

// Timing controls how long Run waits for the log; the zero value is 60 s polled every 250 ms.
type Timing struct {
	Timeout time.Duration
	Poll    time.Duration
}

func (t Timing) withDefaults() Timing {
	t.Timeout = cmp.Or(t.Timeout, 60*time.Second)
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
	out := captureOf(exited)
	if out != nil {
		captured.Delete(exited)
		defer out.release()
	}
	// stalled reports a known cause in the process's output while the game is not up, and ends the process: Wine
	// leaves an assertion dialog open forever, so waiting out the timeout would only hide the reason.
	stalled := func(exited bool) *Failure {
		hint, line, found := Diagnose(out.tail(), exited)
		if !found {
			return nil
		}
		out.stop()
		return &Failure{Hint: hint, Err: errors.New(line)}
	}
	// File timestamps come from a coarse kernel clock that can trail time.Now by a few milliseconds.
	tail := tailer{path: c.LogFile, since: began.Add(-clockSlack), onLines: onLines}
	deadline := time.NewTimer(tm.Timeout)
	defer deadline.Stop()
	tick := time.NewTicker(tm.Poll)
	defer tick.Stop()
	ok := func() bool {
		if c.Ready != nil {
			return c.Ready()
		}
		return tail.poll()
	}
	started := func() error {
		if c.LogFile != "" {
			go follow(ctx, &tail, tm.Poll)
		}
		if c.OnExit != nil && exited != nil && !c.Relay {
			ch := exited
			go func() { c.OnExit(WaitError(<-ch)) }()
		}
		return nil
	}
	for {
		if ok() {
			return started()
		}
		if f := stalled(false); f != nil {
			return f
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return &Failure{Hint: c.Failure, Err: errors.New("the game did not start in time")}
		case waitErr := <-exited:
			if ok() {
				return started()
			}
			if c.Relay && waitErr == nil {
				exited = nil
				continue
			}
			if f := stalled(true); f != nil {
				return f
			}
			return &ExitError{Code: waitCode(waitErr), Output: lastLines(out.tail(), 5)}
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
	// rest is a trailing partial line, held until its newline arrives.
	rest string
}

// poll reports whether the log has been rewritten since the launch, emitting any new lines first.
func (t *tailer) poll() bool {
	if t.path == "" {
		return false
	}
	st, err := os.Stat(t.path)
	if err != nil || !st.ModTime().After(t.since) {
		return false
	}
	f, err := fsx.Open(t.path)
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
