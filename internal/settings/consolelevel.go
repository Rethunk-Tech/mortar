package settings

import (
	"fmt"
	"slices"
	"strings"
)

// consoleShownLevels are the levels a console level list may name.
var consoleShownLevels = []string{ConsoleLevelTrace, ConsoleLevelDebug, ConsoleLevelInfo, ConsoleLevelWarn, ConsoleLevelError, "alert"}

// validConsoleLevel accepts one level (it and above) or a comma list of levels, none twice.
func validConsoleLevel(v string) bool {
	if slices.Contains(consoleLevelValues, v) {
		return true
	}
	levels := strings.Split(v, ",")
	slices.Sort(levels)
	return v != "" && len(slices.Compact(slices.Clone(levels))) == len(levels) &&
		!slices.ContainsFunc(levels, func(l string) bool { return !slices.Contains(consoleShownLevels, l) })
}

func consoleLevelPref() pref {
	return pref{
		spec: PrefSpec{Key: "consoleLevel", Scope: ScopeGame, Type: TypeString},
		get:  func(s Settings, g string) string { return s.GamePrefs(g).ConsoleLevel },
		set: func(s *Settings, g, v string) error {
			if !validConsoleLevel(v) {
				return fmt.Errorf("console level must be a level, or levels joined by commas, got %q", v)
			}
			gp := s.GamePrefs(g)
			gp.ConsoleLevel = v
			putGame(s, g, gp)
			return nil
		},
	}
}
