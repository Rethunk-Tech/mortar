package avscan

import (
	"time"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/nowindow"
)

// command runs the player's own scanner with {path} replaced by the folder: exit 0 is clean, any other code a
// detection whose name is the first line of its output.
type command struct{ line string }

// splitCommand splits a command line on spaces, keeping a double-quoted part whole.
func splitCommand(line string) []string {
	var out []string
	var cur strings.Builder
	quoted, started := false, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted, started = !quoted, true
		case (r == ' ' || r == '\t') && !quoted:
			if started {
				out = append(out, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(r)
			started = true
		}
	}
	if started {
		out = append(out, cur.String())
	}
	return out
}

// commandWaitDelay is how long a cancelled scan waits for the scanner's leftover children to close its output.
const commandWaitDelay = 2 * time.Second

func (c command) Scan(ctx context.Context, dir string) (Detection, bool, error) {
	parts := splitCommand(c.line)
	if len(parts) == 0 {
		return Detection{}, false, ErrNoScanner
	}
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(p, "{path}", dir)
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...) //nolint:gosec // G204: argv is the user's configured scanner command with {path} substituted
	nowindow.Set(cmd)
	// A scanner can leave a child holding stdout open after it is killed; WaitDelay ends the wait for it.
	cmd.WaitDelay = commandWaitDelay
	out, err := cmd.Output()
	if err == nil {
		return Detection{}, false, nil
	}
	if ctx.Err() != nil {
		return Detection{}, false, ctx.Err()
	}
	if _, exited := errors.AsType[*exec.ExitError](err); exited {
		return Detection{Name: firstLine(string(out), filepath.Base(parts[0])), Scanner: filepath.Base(parts[0])}, true, nil
	}
	return Detection{}, false, fmt.Errorf("run %s: %w", parts[0], err)
}

func firstLine(s, fallback string) string {
	for line := range strings.SplitSeq(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return "detected by " + fallback
}
