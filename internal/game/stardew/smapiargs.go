package stardew

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DeniedLaunchOptions are SMAPI flags Mortar always sets itself.
var DeniedLaunchOptions = []string{"--mods-path", "--no-terminal", "--skip-terminal"}

// ParseLaunchOptions splits a profile's extra SMAPI arguments as shell-like words
// (spaces separate; single or double quotes keep spaces; backslash escapes the next rune).
func ParseLaunchOptions(s string) ([]string, error) {
	words, err := splitShellWords(s)
	if err != nil {
		return nil, err
	}
	for _, w := range words {
		flag, _, _ := strings.Cut(w, "=")
		for _, denied := range DeniedLaunchOptions {
			if flag == denied {
				return nil, fmt.Errorf("%s is set by Mortar and cannot be in launch options", denied)
			}
		}
	}
	return words, nil
}

func splitShellWords(s string) ([]string, error) {
	var words []string
	var b strings.Builder
	quote := rune(0)
	escape := false
	started := false
	flush := func() {
		if started {
			words = append(words, b.String())
			b.Reset()
			started = false
		}
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if escape {
			b.WriteRune(r)
			started = true
			escape = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escape = true
			started = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		b.WriteRune(r)
		started = true
	}
	if escape {
		return nil, fmt.Errorf("launch options end with a backslash")
	}
	if quote != 0 {
		return nil, fmt.Errorf("launch options have an unclosed quote")
	}
	flush()
	return words, nil
}

func appendLaunchArgs(goos string, modsArgs, extra []string) []string {
	if goos == "windows" {
		return append(append([]string{}, modsArgs...), extra...)
	}
	return append(append([]string{"--skip-terminal", "--"}, modsArgs...), extra...)
}
