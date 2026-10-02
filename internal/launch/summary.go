package launch

import (
	"regexp"
	"slices"
	"strings"
)

const omittedStartupLog = "[Mortar] Earlier startup log omitted."

// Outcome is how a recorded launch ended.
type Outcome string

const (
	OutcomeRan     Outcome = "ran"
	OutcomeFailed  Outcome = "failed"
	OutcomeCrashed Outcome = "crashed"
)

// MaxLogBytes caps a stored SMAPI log: the console's MaxLines at 256 bytes a line.
const MaxLogBytes = MaxLines * 256

var (
	versionsRe = regexp.MustCompile(`SMAPI (\S+) with Stardew Valley (\S+)`)
	crashRe    = regexp.MustCompile(`(?i)\bfatal\b|\bcrashed\b|game crash`)
)

// ModError is one mod that logged errors in a run, with how many and the first message.
type ModError struct {
	Mod   string `json:"mod"`
	Count int    `json:"count"`
	First string `json:"first"`
}

// Summary is what a SMAPI log says about one run.
type Summary struct {
	SMAPI    string
	Game     string
	Crashed  bool
	Errors   int
	Warnings int
	Mods     []ModError
	ModRefs  []ModRef
}

// ParseLog turns a SMAPI log into entries, hiding the same suppressed messages as the console.
func ParseLog(log string) []Entry {
	var p Parser
	var out []Entry
	for l := range strings.SplitSeq(log, "\n") {
		if l = strings.TrimRight(l, "\r"); l == "" {
			continue
		}
		if e, shown := p.Parse(l); shown {
			out = append(out, e)
		}
	}
	return out
}

// Summarize reads versions, error and warning counts, crash/fatal markers, and per-mod errors from log.
func Summarize(log string) Summary {
	s := Summary{}
	if m := versionsRe.FindStringSubmatch(log); m != nil {
		s.SMAPI, s.Game = m[1], m[2]
	}
	counts := map[string]*ModError{}
	for _, e := range ParseLog(log) {
		if e.Cont {
			continue
		}
		switch e.Level {
		case Warn:
			s.Warnings++
		case Error, Alert:
			s.Errors++
			if e.Mod != "" {
				row, ok := counts[e.Mod]
				if !ok {
					row = &ModError{Mod: e.Mod, First: e.Message}
					counts[e.Mod] = row
				}
				row.Count++
			}
		case Trace, Debug, Info:
		}
		if e.Level == Alert || ((e.Level == Error || e.Level == Alert) && crashRe.MatchString(e.Message)) {
			s.Crashed = true
		}
	}
	s.Mods = make([]ModError, 0, len(counts))
	for _, row := range counts {
		s.Mods = append(s.Mods, *row)
	}
	slices.SortFunc(s.Mods, func(a, b ModError) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Mod, b.Mod)
	})
	return s
}

// CapLog keeps the last max bytes of log, starting at a line boundary when it has to trim.
func CapLog(log string, limit int) string {
	if limit <= 0 || len(log) <= limit {
		return log
	}
	if end := smapiModsLoadedEnd(log); end > 0 && end < len(log) {
		marker := omittedStartupLog + "\n"
		if !strings.HasSuffix(log[:end], "\n") {
			marker = "\n" + marker
		}
		if len(log[:end])+len(marker) < limit {
			tail := tailAtBoundary(log[end:], limit-len(log[:end])-len(marker))
			out := log[:end] + marker + tail
			if len(out) <= limit {
				return out
			}
		}
	}
	return tailAtBoundary(log, limit)
}

func tailAtBoundary(log string, limit int) string {
	if limit <= 0 {
		return ""
	}
	cut := log[len(log)-limit:]
	if i := strings.IndexByte(cut, '\n'); i >= 0 && i+1 < len(cut) {
		return cut[i+1:]
	}
	return cut
}

// FormatLog writes entries the way SMAPI pads a header line.
func FormatLog(entries []Entry) string {
	var b strings.Builder
	for i, e := range entries {
		if i > 0 {
			b.WriteByte('\n')
		}
		if e.Cont || e.Time == "" {
			b.WriteString(e.Message)
			continue
		}
		b.WriteByte('[')
		b.WriteString(e.Time)
		b.WriteByte(' ')
		level := string(e.Level)
		b.WriteString(level)
		for n := len(level); n < 5; n++ {
			b.WriteByte(' ')
		}
		b.WriteByte(' ')
		b.WriteString(e.Mod)
		b.WriteString("] ")
		b.WriteString(e.Message)
	}
	return b.String()
}
