package doctor

import (
	"os"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Replaced by tests.
var (
	environ   = os.Environ
	osRelease = func() []byte {
		b, _ := fsx.ReadFile("/etc/os-release")
		return b
	}
)

// deckChecks reports a Steam Deck or SteamOS host and a gamescope session. Game Mode runs gamescope, where Steam
// owns every launch of a Proton game, so Mortar's own Play only works from Desktop Mode. Neither is a finding elsewhere.
func deckChecks() []Check {
	env := map[string]string{}
	gamescope := false
	for _, kv := range environ() {
		k, v, _ := strings.Cut(kv, "=")
		env[k] = v
		gamescope = gamescope || strings.HasPrefix(k, "GAMESCOPE_")
	}
	gamescope = gamescope || strings.EqualFold(env["XDG_CURRENT_DESKTOP"], "gamescope")
	deck := env["SteamDeck"] == "1" || env["SteamOS"] == "1" || osReleaseID(osRelease()) == "steamos"
	var checks []Check
	if deck {
		c := Check{ID: "steamDeck", Status: Pass, Detail: "Steam Deck or SteamOS: Desktop Mode launches games from Mortar"}
		if gamescope {
			c.Status = Warn
			c.Detail = "Steam Deck Game Mode: Proton games start through Steam here, not from Mortar's Play"
			c.Fix = "switch to Desktop Mode to launch from Mortar, or add the game to Steam and play it from Game Mode"
		}
		checks = append(checks, c)
	}
	if gamescope && !deck {
		checks = append(checks, Check{
			ID:     "gamescope",
			Status: Warn,
			Detail: "gamescope session: Steam starts Proton games here, so a direct launch from Mortar may not show a window",
			Fix:    "launch from a desktop session, or play the game through Steam",
		})
	}
	return checks
}

func osReleaseID(b []byte) string {
	for line := range strings.SplitSeq(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "ID="); ok {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}
