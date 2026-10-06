package game

import (
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// launchTool is the file name, without extension, that identifies a loader's executable inside Steam launch options.
func launchTool(exe string) string {
	base := filepath.Base(exe)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// launchHas reports whether Steam launch options run exe in place of the game: `"<game>\Loader.exe" %command%`.
func launchHas(options, exe string) bool {
	return strings.Contains(strings.ToLower(options), strings.ToLower(launchTool(exe))) && strings.Contains(options, "%command%")
}

// launchWith puts exe before Steam's %command%, or adds the line ahead of options without one.
func launchWith(exe, current string) string {
	if launchHas(current, exe) {
		return current
	}
	quoted := `"` + exe + `"`
	if strings.Contains(current, "%command%") {
		return strings.Replace(current, "%command%", quoted+" %command%", 1)
	}
	return strings.TrimSpace(quoted + " %command% " + current)
}

// launchWithout removes exe before Steam's %command%, keeping the user's other options.
func launchWithout(exe, current string) string {
	if !launchHas(current, exe) {
		if strings.TrimSpace(current) == "%command%" {
			return ""
		}
		return current
	}
	prefix, suffix, ok := strings.Cut(current, "%command%")
	if !ok {
		return current
	}
	closeQuote := strings.LastIndex(prefix, `"`)
	if closeQuote < 0 {
		return current
	}
	openQuote := strings.LastIndex(prefix[:closeQuote], `"`)
	if openQuote < 0 || !strings.Contains(strings.ToLower(prefix[openQuote:closeQuote+1]), strings.ToLower(launchTool(exe))) {
		return current
	}
	result := strings.TrimSpace(prefix[:openQuote] + "%command%" + suffix)
	if result == "%command%" {
		return ""
	}
	return result
}

// steamExe is the executable Steam's launch options start for the game's loader inside dir ("" gives its bare name).
func steamExe(gameID, dir string) (string, bool) {
	for _, l := range Loaders(gameID) {
		if s, ok := l.(loader.SteamExe); ok {
			return s.SteamExe(dir), true
		}
	}
	return "", false
}

// StartsLoader reports whether Steam launch options already start the game's loader; never for a loader Steam does
// not start (BepInEx, which Mortar places before each launch).
func StartsLoader(gameID, options string) (bool, error) {
	if _, err := Require(gameID); err != nil {
		return false, err
	}
	exe, ok := steamExe(gameID, "")
	return ok && launchHas(options, exe), nil
}

// NeedsLaunchOption reports whether a Steam launch loads mods only once Steam's launch options start the loader.
func NeedsLaunchOption(gameID string) (bool, error) {
	if _, err := Require(gameID); err != nil {
		return false, err
	}
	_, ok := steamExe(gameID, "")
	return ok, nil
}
