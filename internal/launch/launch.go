// Package launch starts a game, decides whether the launch worked, and finds and stops the running loader process.
package launch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/steam"
)

// Hint names why a launch failed, for the frontend to explain in the user's language.
type Hint string

const (
	HintNone Hint = ""
	// HintSteam: the game never wrote its log, so Steam may not be running or signed in.
	HintSteam Hint = "steam"
	// HintLaunchOptions: Steam's launch options for the game lack the loader's line.
	HintLaunchOptions Hint = "launch-options"
)

// ErrNoSteam means there is no Steam to launch through; the user may choose to launch directly.
var ErrNoSteam = errors.New("no Steam was found")

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
	// Steam is nil when no Steam was found.
	Steam *steam.Steam
	// Direct launches the loader without Steam, after the user agreed to lose the overlay and playtime.
	Direct bool
}

// Runner starts a command and returns without waiting for it to finish.
type Runner func(dir, name string, args ...string) error

// Start is the Runner that runs the real command, reaping it in the background.
func Start(dir, name string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Command is one process to start, plus how to tell it worked.
type Command struct {
	Dir  string
	Name string
	Args []string
	// LogFile is the file the game rewrites when it starts.
	LogFile string
	// Failure names the hint to give when the log never appears.
	Failure Hint
}

const clockSlack = 50 * time.Millisecond

// Timing controls how long Run waits for the log; the zero value is 60 s polled every 250 ms.
type Timing struct {
	Timeout time.Duration
	Poll    time.Duration
}

func (t Timing) withDefaults() Timing {
	if t.Timeout == 0 {
		t.Timeout = 60 * time.Second
	}
	if t.Poll == 0 {
		t.Poll = 250 * time.Millisecond
	}
	return t
}

// Run starts c and waits for its log file to be rewritten after the start, sending each batch of new log lines to
// onLines as it appears. It returns a *Failure when the log does not change within the timeout. Once the game has
// started, the log keeps being followed until ctx is done, with one last read then; ctx must outlive the game.
func Run(ctx context.Context, run Runner, c Command, tm Timing, onLines func([]string)) error {
	tm = tm.withDefaults()
	began := time.Now()
	if err := run(c.Dir, c.Name, c.Args...); err != nil {
		return &Failure{Hint: c.Failure, Err: err}
	}
	// File timestamps come from a coarse kernel clock that can trail time.Now by a few milliseconds.
	tail := tailer{path: c.LogFile, since: began.Add(-clockSlack), onLines: onLines}
	deadline := time.NewTimer(tm.Timeout)
	defer deadline.Stop()
	tick := time.NewTicker(tm.Poll)
	defer tick.Stop()
	for {
		if tail.poll() {
			go follow(ctx, &tail, tm.Poll)
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return &Failure{Hint: c.Failure, Err: errors.New("the game did not start in time")}
		case <-tick.C:
		}
	}
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
	st, err := os.Stat(t.path)
	if err != nil || !st.ModTime().After(t.since) {
		return false
	}
	f, err := os.Open(t.path)
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
	if len(batch) > 0 {
		t.onLines(batch)
	}
	return true
}
