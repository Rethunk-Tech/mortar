package launchsvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestRunEndNotificationTextClosed(t *testing.T) {
	if _, _, ok := RunEndNotificationText("Stardew Valley", launch.Summary{}); ok {
		t.Fatal("a normal close sends a notification")
	}
}

func TestRunEndNotificationTextCrash(t *testing.T) {
	title, body, ok := RunEndNotificationText("Stardew Valley", launch.Summary{
		Crashed: true,
		Mods:    []launch.ModError{{Mod: "A"}, {Mod: "B"}},
	})
	if !ok || title != "Stardew Valley crashed" || body != "2 mods logged errors" {
		t.Fatalf("got %q / %q", title, body)
	}
	title, body, _ = RunEndNotificationText("Stardew Valley", launch.Summary{Crashed: true, Mods: []launch.ModError{{Mod: "A"}}})
	if title != "Stardew Valley crashed" || body != "1 mod logged errors" {
		t.Fatalf("singular: got %q / %q", title, body)
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
