package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

func TestGameLaunchPresets(t *testing.T) {
	listed := []settings.LaunchPreset{{Name: "Proton", Options: "-d", Prefix: "gamemoderun", Env: "A=1"}}
	r := invoke(t, map[string]any{"game.launchPresets": listed}, "game", "launch-presets", "stardew")
	if r.code != 0 || r.calls[0].method != "game.launchPresets" || r.calls[0].params.Sub != "list" {
		t.Fatalf("list: %+v", r)
	}
	if !strings.Contains(r.out, "Proton") || !strings.Contains(r.out, "-d") {
		t.Fatalf("list out: %q", r.out)
	}
	r = invoke(t, map[string]any{"game.launchPresets": listed},
		"game", "launch-presets", "stardew", "add", "Proton", "-d", "gamemoderun", "A=1")
	if r.code != 0 || r.calls[0].params.Sub != "add" || r.calls[0].params.Name != "Proton" {
		t.Fatalf("add: %+v", r)
	}
	if r.calls[0].params.Value != "-d" || r.calls[0].params.Path != "gamemoderun" || r.calls[0].params.Query != "A=1" {
		t.Fatalf("add fields: %+v", r.calls[0].params)
	}
	r = invoke(t, map[string]any{"game.launchPresets": []settings.LaunchPreset{}},
		"game", "launch-presets", "stardew", "remove", "Proton")
	if r.code != 0 || r.calls[0].params.Sub != "remove" || r.calls[0].params.Name != "Proton" {
		t.Fatalf("remove: %+v", r)
	}
}

func TestPlayTest(t *testing.T) {
	r := invoke(t, map[string]any{"play.test": map[string]any{"reachedTitle": true}},
		"play", "stardew", "Farm", "--test")
	if r.code != 0 || r.calls[0].method != "play.test" || !strings.Contains(r.out, "Reached the title screen") {
		t.Fatalf("title: %+v %q", r, r.out)
	}
	r = invoke(t, map[string]any{"play.test": map[string]any{"cause": "SIGABRT"}},
		"play", "stardew", "Farm", "--test")
	if r.code != 0 || !strings.Contains(r.out, "Crashed: SIGABRT") {
		t.Fatalf("crash: %+v %q", r, r.out)
	}
}
