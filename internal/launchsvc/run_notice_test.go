package launchsvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestRunEndNotificationText(t *testing.T) {
	for _, c := range []struct {
		name    string
		summary launch.Summary
		send    bool
		title   string
		body    string
	}{
		{"normal close", launch.Summary{}, false, "", ""},
		{"mods", launch.Summary{Crashed: true, Mods: []launch.ModError{{Mod: "A"}, {Mod: "B"}}}, true, "Stardew Valley crashed", "Mods with errors: 2"},
		{"one mod", launch.Summary{Crashed: true, Mods: []launch.ModError{{Mod: "A"}}}, true, "Stardew Valley crashed", "Mods with errors: 1"},
		{"exit code", launch.Summary{Crashed: true, Exit: launch.Exit{Code: 134, Signal: "SIGABRT"}}, true, "Stardew Valley crashed", "Exited with code 134 (SIGABRT)"},
	} {
		title, body, ok := RunEndNotificationText("Stardew Valley", c.summary)
		if ok != c.send || (ok && (title != c.title || body != c.body)) {
			t.Errorf("%s: got %q / %q, %v", c.name, title, body, ok)
		}
	}
}

func TestNoticeProfileFromResponse(t *testing.T) {
	g, p, tab := NoticeProfileFromResponse("run-end-abc", map[string]any{"game": "stardew", "profile": "prof"})
	if g != "stardew" || p != "prof" || tab != "console" {
		t.Fatalf("run-end: got %q %q tab %q", g, p, tab)
	}
	g, p, tab = NoticeProfileFromResponse("mod-updates-prof-123", map[string]any{"game": "stardew", "profile": "prof"})
	if g != "stardew" || p != "prof" || tab != "updates" {
		t.Fatalf("mod-updates: got %q %q tab %q", g, p, tab)
	}
	if g, p, tab := NoticeProfileFromResponse("nxm-1", map[string]any{"profile": "prof"}); g != "" || p != "" || tab != "" {
		t.Fatalf("wrong id: got %q %q tab %q", g, p, tab)
	}
}
