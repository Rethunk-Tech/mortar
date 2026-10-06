package launch

import (
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
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

var loadedSaveRe = regexp.MustCompile(`(?i)(?:Context:\s*)?loaded save '([^']+)'`)

// LoadedSave returns the save folder SMAPI loaded, from the last matching SMAPI line in log.
func LoadedSave(log string) (string, bool) {
	folder := ""
	for _, e := range ParseLog(log) {
		if e.Mod != "SMAPI" {
			continue
		}
		if m := loadedSaveRe.FindStringSubmatch(e.Message); m != nil {
			folder = m[1]
		}
	}
	return folder, folder != ""
}

// ModsPath returns the mods folder SMAPI's log says it loaded, as SMAPI wrote it, and false when the log does not
// say. SMAPI writes the home folder as ~ (`PathUtilities.AnonymizePathForDisplay`).
func ModsPath(log string) (string, bool) {
	for l := range strings.Lines(log) {
		m := header.FindStringSubmatch(strings.TrimRight(l, "\r\n"))
		if m == nil || strings.TrimSpace(m[3]) != "SMAPI" {
			continue
		}
		if path, ok := strings.CutPrefix(m[4], "Mods go here: "); ok {
			return path, true
		}
	}
	return "", false
}

// LogOwnedBy reports whether log was written by SMAPI loading modsDir. home is the folder SMAPI shortened to ~.
// A log that does not name its mods folder belongs to no one.
func LogOwnedBy(log, home, modsDir string) bool {
	path, ok := ModsPath(log)
	if !ok {
		return false
	}
	if rest, ok := strings.CutPrefix(path, "~"); ok && (rest == "" || rest[0] == '/' || rest[0] == '\\') {
		path = home + rest
	}
	path, modsDir = filepath.Clean(path), filepath.Clean(modsDir)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(path, modsDir)
	}
	return path == modsDir
}

// Parser turns log lines into entries, remembering the last header so continuation lines can inherit it.
type Parser struct {
	last   Entry
	seen   bool
	hidden bool
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

// Parse returns the entry for one SMAPI or BepInEx line, and false when the line belongs to a suppressed message. A line that is not
// a header continues the previous entry; with none yet it stands alone as an INFO line.
func (p *Parser) Parse(line string) (Entry, bool) {
	if m := header.FindStringSubmatch(line); m != nil {
		p.last = Entry{Time: m[1], Level: Level(m[2]), Mod: strings.TrimSpace(m[3]), Message: m[4]}
		p.seen = true
		p.hidden = slices.Contains(suppressed, p.last.Message)
		return p.last, !p.hidden
	}
	if e, ok := parseBepInEx(line); ok {
		p.last, p.seen, p.hidden = e, true, false
		return e, true
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
