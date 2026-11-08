package game

import (
	"io"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// GameLogText is the text of the game's own log files for roles, in the order given, each the last
// launch.MaxLogBytes from a line start: a game's log can grow without bound and its analyzers only need the latest
// run. A role the game keeps no path for, or a file it has not written, adds nothing.
func GameLogText(home string, set settings.Settings, id string, roles ...string) string {
	var b strings.Builder
	for _, role := range roles {
		if !HasPath(id, role) {
			continue
		}
		path, err := PathFor(home, set, id, "", role)
		if err != nil {
			continue
		}
		text := readLogTail(path)
		if text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(text)
	}
	return b.String()
}

// readLogTail is the last launch.MaxLogBytes of the log at path from a line start, empty when it cannot be read.
func readLogTail(path string) string {
	f, err := safefs.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return ""
	}
	trimmed := st.Size() > launch.MaxLogBytes
	if trimmed {
		if _, err := f.Seek(-int64(launch.MaxLogBytes), io.SeekEnd); err != nil {
			return ""
		}
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	text := string(b)
	if trimmed {
		_, text, _ = strings.Cut(text, "\n")
	}
	return strings.ToValidUTF8(text, "")
}
