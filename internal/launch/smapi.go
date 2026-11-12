package launch

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-smapi/logline"
)

// Level is a console log level: SMAPI's own, which BepInEx's levels map onto.
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

// Entry is one line of SMAPI's or BepInEx's log; a BepInEx line has no Time and names its source in Mod. A line that
// continues an earlier entry's multi-line message repeats that entry's time, level and mod and has Cont set, so
// filters keep a message's lines together.
type Entry struct {
	// Seq orders entries across the process, letting a reader that fetched history skip lines it already has.
	Seq     int64  `json:"seq"`
	Time    string `json:"time"`
	Level   Level  `json:"level"`
	Mod     string `json:"mod"`
	Message string `json:"message"`
	Cont    bool   `json:"cont"`
	// Code and Params let the window word one of Mortar's own lines in the user's language; Message stays the English
	// line the log files and the CLI carry. Lines from the game or a mod have no Code.
	Code   string            `json:"code,omitempty"`
	Params map[string]string `json:"params,omitempty"`
	// Load marks a loader's line for a mod it is loading, which the launch screen counts rather than shows.
	Load bool `json:"load,omitempty"`
	// Source is the method that logged a game line through Unity (Type::Method), and Object the hierarchy
	// path of the object it named, from the tag the BepInEx bridge starts the line with.
	Source string `json:"source,omitempty"`
	Object string `json:"object,omitempty"`
}

// Parser turns log lines into entries, remembering the last header so continuation lines can inherit it.
type Parser struct {
	// Now, when set, times a header its loader wrote without one (stock BepInEx) as it arrives, so a live console can
	// show when each line came. A log read back from disk leaves it nil: its arrival time is not known.
	Now    func() time.Time
	last   Entry
	seen   bool
	hidden bool
}

// arrivalStamp is the shape AsyncLoggers stamps BepInEx lines with: never SMAPI's bare HH:MM:SS, which FormatLog
// would write back as a SMAPI header.
const arrivalStamp = "15:04:05.0000000"

func (p *Parser) header(e Entry) Entry {
	if e.Time == "" && p.Now != nil {
		e.Time = p.Now().Format(arrivalStamp)
	}
	p.last, p.seen = e, true
	return e
}

func smapiModsLoaded(e Entry) bool {
	message := strings.ToLower(e.Message)
	return e.Mod == "SMAPI" && strings.Contains(message, "loaded") && strings.Contains(message, "mod")
}

func smapiModsLoadedEnd(log string) int {
	var p Parser
	offset := 0
	for line := range strings.Lines(log) {
		end := offset + len(line)
		trimmed := strings.TrimRight(line, "\r\n")
		if e, shown := p.Parse(trimmed); shown && smapiModsLoaded(e) {
			return end
		}
		offset = end
	}
	return 0
}

// smapiHeader reads line as a SMAPI header, and false for a continuation or a line that is not one.
func smapiHeader(line string) (Entry, bool) {
	var p logline.Parser
	e := p.Parse(line)
	if e.Time == "" || e.Cont {
		return Entry{}, false
	}
	return Entry{Time: e.Time, Level: Level(e.Level), Mod: e.Mod, Message: e.Message}, true
}

// Parse returns the entry for one SMAPI or BepInEx line, and false when the line belongs to a hidden message (SetHiddenMessages). A line that is not
// a header continues the previous entry; with none yet it stands alone as an INFO line.
func (p *Parser) Parse(line string) (Entry, bool) {
	if e, ok := smapiHeader(line); ok {
		p.header(e)
		p.hidden = isHidden(p.last.Mod, p.last.Message)
		return p.last, !p.hidden
	}
	if e, ok := parseBepInEx(line); ok {
		p.hidden = false
		return p.header(e), true
	}
	if e, ok := parseVintageStory(line); ok {
		p.hidden = false
		return p.header(e), true
	}
	for _, parse := range []func(string) (Entry, bool){parseLovely, parseSerilog, parseROM, parseNorthstar} {
		if e, ok := parse(line); ok {
			p.hidden = false
			return p.header(e), true
		}
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
	mu         sync.Mutex
	lines      []Entry
	head       []Entry
	tail       []Entry
	summarized bool
	// Cap overrides MaxLines when greater than zero.
	Cap int
}

func (b *Buffer) cap() int {
	if b.Cap > 0 {
		return b.Cap
	}
	return MaxLines
}

// Add appends e, discarding the oldest entries beyond MaxLines. It compacts only once a quarter over the bound so
// the copy is amortised.
func (b *Buffer) Add(e Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.summarized {
		b.tail = append(b.tail, e)
		b.trimTail()
		return
	}
	b.lines = append(b.lines, e)
	if smapiModsLoaded(e) {
		b.head = slices.Clone(b.lines)
		b.lines = nil
		b.summarized = true
		b.trimHead()
		return
	}
	limit := b.cap()
	if len(b.lines) > limit+limit/4 {
		b.lines = slices.Clone(b.lines[len(b.lines)-limit:])
	}
}

// Lines returns a copy of the newest MaxLines entries, oldest first.
func (b *Buffer) Lines() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.summarized {
		out := make([]Entry, 0, len(b.head)+len(b.tail)+1)
		out = append(out, b.head...)
		if len(b.tail) > 0 {
			out = append(out, Entry{Level: Info, Mod: "Mortar", Message: omittedStartupLog})
			out = append(out, b.tail...)
		}
		return out
	}
	return slices.Clone(b.lines[max(0, len(b.lines)-b.cap()):])
}

func (b *Buffer) trimHead() {
	limit := b.cap()
	if len(b.head) >= limit {
		b.head = slices.Clone(b.head[len(b.head)-limit+1:])
	}
	b.trimTail()
}

func (b *Buffer) trimTail() {
	limit := max(0, b.cap()-len(b.head)-1)
	if len(b.tail) > limit {
		b.tail = slices.Clone(b.tail[len(b.tail)-limit:])
	}
}
