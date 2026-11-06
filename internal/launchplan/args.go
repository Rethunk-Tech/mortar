package launchplan

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseArgs splits a profile's extra launch arguments as shell-like words (spaces separate; single or double quotes
// keep spaces; backslash escapes the next rune) and refuses a flag in denied, which the loader sets itself. On Windows
// a backslash is a path separator and only double quotes quote, as on its command line: `\"` is a literal quote.
func ParseArgs(s string, denied []string) ([]string, error) {
	words, err := splitShellWords(s, runtime.GOOS == "windows")
	if err != nil {
		return nil, err
	}
	for _, w := range words {
		flag, _, _ := strings.Cut(w, "=")
		for _, d := range denied {
			if flag == d {
				return nil, fmt.Errorf("%s is set by Mortar and cannot be in launch options", d)
			}
		}
	}
	return words, nil
}

// shellSplitter holds the state of one splitShellWords pass.
type shellSplitter struct {
	s       string
	i       int
	windows bool
	words   []string
	b       strings.Builder
	quote   rune
	escape  bool
	started bool
}

func (p *shellSplitter) flush() {
	if p.started {
		p.words = append(p.words, p.b.String())
		p.b.Reset()
		p.started = false
	}
}

func (p *shellSplitter) literal(r rune) {
	p.b.WriteRune(r)
	p.started = true
}

// backslashRun handles a Windows backslash run. A run of n backslashes before a quote is n/2 backslashes, and an
// odd n makes the quote literal; the quote of an even run toggles quoting. Anywhere else backslashes are literal.
func (p *shellSplitter) backslashRun() {
	run := 1
	for strings.HasPrefix(p.s[p.i:], `\`) {
		run++
		p.i++
	}
	p.started = true
	if !strings.HasPrefix(p.s[p.i:], `"`) {
		p.b.WriteString(strings.Repeat(`\`, run))
		return
	}
	p.b.WriteString(strings.Repeat(`\`, run/2))
	p.i++
	switch {
	case run%2 == 1:
		p.b.WriteByte('"')
	case p.quote == '"':
		p.quote = 0
	default:
		p.quote = '"'
	}
}

func (p *shellSplitter) step(r rune) {
	switch {
	case p.escape:
		p.literal(r)
		p.escape = false
	case r == '\\' && p.windows:
		p.backslashRun()
	case r == '\\' && p.quote != '\'':
		p.escape = true
		p.started = true
	case p.quote != 0:
		if r == p.quote {
			p.quote = 0
		} else {
			p.b.WriteRune(r)
		}
	case r == '"' || (r == '\'' && !p.windows):
		p.quote = r
		p.started = true
	case unicode.IsSpace(r):
		p.flush()
	default:
		p.literal(r)
	}
}

func splitShellWords(s string, windows bool) ([]string, error) {
	p := &shellSplitter{s: s, windows: windows}
	for p.i < len(s) {
		r, size := utf8.DecodeRuneInString(s[p.i:])
		p.i += size
		p.step(r)
	}
	if p.escape {
		return nil, fmt.Errorf("launch options end with a backslash")
	}
	if p.quote != 0 {
		return nil, fmt.Errorf("launch options have an unclosed quote")
	}
	p.flush()
	return p.words, nil
}
