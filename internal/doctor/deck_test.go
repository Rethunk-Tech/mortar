package doctor

import "testing"

func TestDeckChecks(t *testing.T) {
	var env []string
	rel := ""
	prevEnv, prevRel := environ, osRelease
	environ = func() []string { return env }
	osRelease = func() []byte { return []byte(rel) }
	t.Cleanup(func() {
		environ, osRelease = prevEnv, prevRel
	})
	run := func() map[string]string {
		out := map[string]string{}
		for _, c := range deckChecks() {
			out[c.ID] = c.Status
		}
		return out
	}
	if got := run(); len(got) != 0 {
		t.Fatalf("plain desktop: %v", got)
	}
	rel = "NAME=SteamOS\nID=steamos\n"
	if got := run(); got["steamDeck"] != Pass || len(got) != 1 {
		t.Fatalf("SteamOS desktop: %v", got)
	}
	rel, env = "", []string{"SteamDeck=1", "GAMESCOPE_WAYLAND_DISPLAY=gamescope-0"}
	if got := run(); got["steamDeck"] != Warn || len(got) != 1 {
		t.Fatalf("Deck Game Mode: %v", got)
	}
	env = []string{"XDG_CURRENT_DESKTOP=gamescope"}
	if got := run(); got["gamescope"] != Warn || len(got) != 1 {
		t.Fatalf("gamescope off Deck: %v", got)
	}
}
