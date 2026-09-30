package launch

import (
	"regexp"
	"slices"
	"strings"
	"sync"
)

// Level is a SMAPI log level.
type Level string

const (
	Trace Level = "TRACE"
	Debug Level = "DEBUG"
	Info  Level = "INFO"
	Warn  Level = "WARN"
	Error Level = "ERROR"
	Alert Level = "ALERT"
)

// MaxLines bounds a Buffer.
const MaxLines = 20000

// Entry is one line of SMAPI's log. A line that continues an earlier entry's multi-line message repeats that
// entry's time, level and mod and has Cont set, so filters keep a message's lines together.
type Entry struct {
	// Seq orders entries across the process, letting a reader that fetched history skip lines it already has.
	Seq     int64  `json:"seq"`
	Time    string `json:"time"`
	Level   Level  `json:"level"`
	Mod     string `json:"mod"`
	Message string `json:"message"`
	Cont    bool   `json:"cont"`
}

// SMAPI writes `[HH:MM:SS LEVEL  Mod] message`, padding the level to five characters.
var header = regexp.MustCompile(`^\[(\d\d:\d\d:\d\d) (TRACE|DEBUG|INFO|WARN|ERROR|ALERT) *([^\]]*)\] ?(.*)$`)

// suppressed lists log messages Mortar never shows, with their continuation lines. Mortar passes --no-terminal on
// purpose and its Console replaces the terminal, so SMAPI's complaint about having none only misleads. The game's
// own GOG Galaxy start-up fails on Steam launches and reads like a broken mod when it is neither.
var suppressed = []string{
	"Writing to the terminal is disabled because the --no-terminal argument was received. This usually means launching the terminal failed.",
	"Error initializing the Galaxy API.",
	"Galaxy SignInSteam failed with an exception:",
}

// Parser turns log lines into entries, remembering the last header so continuation lines can inherit it.
type Parser struct {
	last   Entry
	seen   bool
	hidden bool
}

// Parse returns the entry for one line, and false when the line belongs to a suppressed message. A line that is not
// a header continues the previous entry; with none yet it stands alone as an INFO line.
func (p *Parser) Parse(line string) (Entry, bool) {
	if m := header.FindStringSubmatch(line); m != nil {
		p.last = Entry{Time: m[1], Level: Level(m[2]), Mod: strings.TrimSpace(m[3]), Message: m[4]}
		p.seen = true
		p.hidden = slices.Contains(suppressed, p.last.Message)
		return p.last, !p.hidden
	}
	if !p.seen {
		return Entry{Level: Info, Message: line}, true
	}
	e := p.last
	e.Message, e.Cont = line, true
	return e, !p.hidden
}

// Buffer keeps the newest MaxLines entries.
type Buffer struct {
	mu    sync.Mutex
	lines []Entry
}

// Add appends e, discarding the oldest entries beyond MaxLines. It compacts only once a quarter over the bound so
// the copy is amortised.
func (b *Buffer) Add(e Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, e)
	if len(b.lines) > MaxLines+MaxLines/4 {
		b.lines = slices.Clone(b.lines[len(b.lines)-MaxLines:])
	}
}

// Lines returns a copy of the newest MaxLines entries, oldest first.
func (b *Buffer) Lines() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.lines[max(0, len(b.lines)-MaxLines):])
}
