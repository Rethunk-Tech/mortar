package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

func TestGameLaunchPresetTemplates(t *testing.T) {
	listed := []settings.LaunchPresetTemplate{{Name: "Proton", Options: "-d", Prefix: "gamemoderun", Env: "A=1"}}
	r := invoke(t, map[string]any{"game.launchPresetTemplates": listed}, "game", "launch-preset-templates", "stardew")
	if r.code != 0 || r.calls[0].method != "game.launchPresetTemplates" || r.calls[0].params.Sub != "list" {
		t.Fatalf("list: %+v", r)
	}
	if !strings.Contains(r.out, "Proton") || !strings.Contains(r.out, "-d") {
		t.Fatalf("list out: %q", r.out)
	}
	r = invoke(t, map[string]any{"game.launchPresetTemplates": listed},
		"game", "launch-preset-templates", "stardew", "add", "Proton", "-d", "gamemoderun", "A=1")
	if r.code != 0 || r.calls[0].params.Sub != "add" || r.calls[0].params.Name != "Proton" {
		t.Fatalf("add: %+v", r)
	}
	if r.calls[0].params.Value != "-d" || r.calls[0].params.Path != "gamemoderun" || r.calls[0].params.Query != "A=1" {
		t.Fatalf("add fields: %+v", r.calls[0].params)
	}
	r = invoke(t, map[string]any{"game.launchPresetTemplates": []settings.LaunchPresetTemplate{}},
		"game", "launch-preset-templates", "stardew", "remove", "Proton")
	if r.code != 0 || r.calls[0].params.Sub != "remove" || r.calls[0].params.Name != "Proton" {
		t.Fatalf("remove: %+v", r)
	}
	r = invoke(t, map[string]any{"game.launchPresetTemplates": listed},
		"game", "launch-preset-templates", "stardew", "use", "Proton", "Farm")
	if r.code != 0 || r.calls[0].params.Sub != "use" || r.calls[0].params.Profile != "Farm" || r.calls[0].params.Name != "Proton" {
		t.Fatalf("use: %+v", r)
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
