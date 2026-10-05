package settings

import (
	"reflect"
	"testing"
)

func TestLaunchPresetCRUD(t *testing.T) {
	s, _ := open(t)
	if got := s.ListLaunchPresetTemplates("stardew"); len(got) != 0 {
		t.Fatalf("empty list = %#v", got)
	}
	first := LaunchPresetTemplate{Name: "  Proton  ", Options: "--developer-mode", Prefix: "gamemoderun", Env: "MANGOHUD=1", ShowConsole: "false"}
	if err := s.AddLaunchPresetTemplate("stardew", first); err != nil {
		t.Fatal(err)
	}
	got := s.ListLaunchPresetTemplates("stardew")
	want := LaunchPresetTemplate{Name: "Proton", Options: first.Options, Prefix: first.Prefix, Env: first.Env, ShowConsole: "false"}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("after add = %#v", got)
	}
	if err := s.AddLaunchPresetTemplate("stardew", LaunchPresetTemplate{Name: "Bad", ShowConsole: "maybe"}); err == nil {
		t.Fatal("an invalid console choice should be refused")
	}
	updated := LaunchPresetTemplate{Name: "proton", Options: "-foo", Prefix: "mangohud", Env: "A=1"}
	if err := s.AddLaunchPresetTemplate("stardew", updated); err != nil {
		t.Fatal(err)
	}
	got = s.ListLaunchPresetTemplates("stardew")
	if len(got) != 1 || got[0].Options != "-foo" || got[0].Name != "proton" {
		t.Fatalf("replace same name = %#v", got)
	}
	if err := s.AddLaunchPresetTemplate("stardew", LaunchPresetTemplate{Name: "Direct"}); err != nil {
		t.Fatal(err)
	}
	if got = s.ListLaunchPresetTemplates("stardew"); len(got) != 2 {
		t.Fatalf("two presets = %#v", got)
	}
	if err := s.RemoveLaunchPresetTemplate("stardew", "PROTON"); err != nil {
		t.Fatal(err)
	}
	got = s.ListLaunchPresetTemplates("stardew")
	if len(got) != 1 || got[0].Name != "Direct" {
		t.Fatalf("after remove = %#v", got)
	}
	if err := s.RemoveLaunchPresetTemplate("stardew", "missing"); err == nil {
		t.Fatal("expected missing preset error")
	}
	if err := s.AddLaunchPresetTemplate("stardew", LaunchPresetTemplate{Name: "  "}); err == nil {
		t.Fatal("expected empty name error")
	}
}

func TestLaunchPresetPersists(t *testing.T) {
	s, _ := open(t)
	preset := LaunchPresetTemplate{Name: "Keep", Options: "-x", Prefix: "p", Env: "E=1"}
	if err := s.AddLaunchPresetTemplate("stardew", preset); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	got := s2.ListLaunchPresetTemplates("stardew")
	if len(got) != 1 || !reflect.DeepEqual(got[0], preset) {
		t.Fatalf("reloaded = %#v", got)
	}
}
