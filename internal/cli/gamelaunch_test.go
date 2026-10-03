package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/control"
)

func TestGameSteamLaunchOption(t *testing.T) {
	r := invoke(t, map[string]any{
		"game.steamLaunchOption": control.SteamLaunchOptionResult{Options: "old"},
	}, "game", "steam-launch-option", "stardew")
	if r.code != 0 || r.calls[0].method != "game.steamLaunchOption" || !strings.Contains(r.out, "old") {
		t.Fatalf("read: %+v %q", r, r.out)
	}
	r = invoke(t, map[string]any{
		"game.steamLaunchOption": control.SteamLaunchOptionResult{Options: "new", Set: true},
	}, "game", "steam-launch-option", "stardew", "--set")
	if r.code != 0 || !r.calls[0].params.Set || !strings.Contains(r.out, "new") {
		t.Fatalf("set: %+v %q", r, r.out)
	}
}
