package launch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// staleCapture is how old an orphaned capture file must be before a later launch sweeps it.
const staleCapture = 24 * time.Hour

// outputKeep bounds how much of a launched process's output is read back to explain a failure.
const outputKeep = 16 << 10

// capture is a launched process's stdout and stderr, written to a file the process inherits, so a game that keeps
// logging after the launch never blocks on a pipe Mortar stopped reading.
type capture struct {
	file *os.File
	cmd  *exec.Cmd
	// released is set before the first unlink attempt, so whichever of release and the process's exit comes second
	// removes the file: Windows refuses to delete a file the still-running child holds open.
	released atomic.Bool
}

// sweepStale removes capture files a crashed or killed Mortar left behind.
func sweepStale() {
	names, _ := filepath.Glob(filepath.Join(os.TempDir(), "mortar-launch-*.log"))
	for _, n := range names {
		if st, err := os.Stat(n); err == nil && time.Since(st.ModTime()) > staleCapture {
			_ = os.Remove(n)
		}
	}
}

// removeIfReleased unlinks the file once the process holding it has exited.
func (c *capture) removeIfReleased() {
	if c != nil && c.released.Load() {
		_ = os.Remove(c.file.Name())
	}
}

// captured maps the channel StartWithEnv returned to its capture, so Run can read why a process ended or stalled.
var captured sync.Map

func newCapture(cmd *exec.Cmd) *capture {
	sweepStale()
	f, err := os.CreateTemp("", "mortar-launch-*.log")
	if err != nil {
		return nil
	}
	cmd.Stdout, cmd.Stderr = f, f
	return &capture{file: f, cmd: cmd}
}

// tail is the end of what the process has written so far.
func (c *capture) tail() string {
	if c == nil {
		return ""
	}
	st, err := c.file.Stat()
	if err != nil || st.Size() == 0 {
		return ""
	}
	size := min(st.Size(), outputKeep)
	buf := make([]byte, size)
	n, _ := c.file.ReadAt(buf, st.Size()-size)
	return strings.ToValidUTF8(string(buf[:n]), "")
}

// release closes and unlinks the file; where the unlink is refused because the process still holds the file, the
// process's exit does it.
func (c *capture) release() {
	if c == nil {
		return
	}
	c.released.Store(true)
	_ = c.file.Close()
	_ = os.Remove(c.file.Name())
}

// stop ends the launched process and everything it started.
func (c *capture) stop() {
	if c != nil && c.cmd.Process != nil {
		killTree(c.cmd.Process.Pid)
	}
}

func captureOf(exited <-chan error) *capture {
	v, ok := captured.Load(exited)
	if !ok {
		return nil
	}
	c, _ := v.(*capture)
	return c
}

// lastLines is the last n non-empty lines of text.
func lastLines(text string, n int) []string {
	var lines []string
	for l := range strings.SplitSeq(text, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines[max(len(lines)-n, 0):]
}

// Diagnose names the known reason a launch cannot come up from the process's output: the Hint for the frontend's
// plain fix and the line that shows it. ok is false when the output holds no known cause. While the process still
// runs only causes that never appear in a healthy start count; a missing file is only read as the cause once it exited.
func Diagnose(output string, exited bool) (hint Hint, line string, ok bool) {
	for _, l := range lastLines(output, 200) {
		low := strings.ToLower(l)
		switch {
		case strings.Contains(low, "unable to load native steamclient"):
			return HintSteamClient, l, true
		case strings.Contains(low, "_wassert") || strings.Contains(low, "assertion failed"):
			return HintWine, l, true
		case exited && (strings.Contains(low, "no such file or directory") || strings.Contains(low, "cannot find the file") || strings.Contains(low, "executable file not found")):
			return HintMissingExe, l, true
		}
	}
	return HintNone, "", false
}
