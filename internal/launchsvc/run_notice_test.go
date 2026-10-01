package launchsvc

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

func TestRunEndNotificationTextClosed(t *testing.T) {
	title, body := RunEndNotificationText("Stardew Valley", launch.Summary{})
	if title != "Game closed" || body != "" {
		t.Fatalf("got %q / %q", title, body)
	}
}

func TestRunEndNotificationTextCrash(t *testing.T) {
	title, body := RunEndNotificationText("Stardew Valley", launch.Summary{
		Crashed: true,
		Mods:    []launch.ModError{{Mod: "A"}, {Mod: "B"}},
	})
	if title != "Stardew Valley crashed" || body != "2 mods logged errors" {
		t.Fatalf("got %q / %q", title, body)
	}
	title, body = RunEndNotificationText("Stardew Valley", launch.Summary{Crashed: true, Mods: []launch.ModError{{Mod: "A"}}})
	if title != "Stardew Valley crashed" || body != "1 mod logged errors" {
		t.Fatalf("singular: got %q / %q", title, body)
	}
}

func TestNoticeProfileFromResponse(t *testing.T) {
	g, p := NoticeProfileFromResponse("run-end-abc", map[string]any{"game": "stardew", "profile": "prof"})
	if g != "stardew" || p != "prof" {
		t.Fatalf("got %q %q", g, p)
	}
	if g, p := NoticeProfileFromResponse("nxm-1", map[string]any{"profile": "prof"}); g != "" || p != "" {
		t.Fatalf("wrong id: got %q %q", g, p)
	}
}
