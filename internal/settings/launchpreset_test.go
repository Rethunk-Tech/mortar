package settings

import (
	"reflect"
	"testing"
)

func TestLaunchPresetCRUD(t *testing.T) {
	s, _ := open(t)
	if got := s.ListLaunchPresets(GameStardew); len(got) != 0 {
		t.Fatalf("empty list = %#v", got)
	}
	first := LaunchPreset{Name: "  Proton  ", Options: "--developer-mode", Prefix: "gamemoderun", Env: "MANGOHUD=1"}
	if err := s.AddLaunchPreset(GameStardew, first); err != nil {
		t.Fatal(err)
	}
	got := s.ListLaunchPresets(GameStardew)
	want := LaunchPreset{Name: "Proton", Options: first.Options, Prefix: first.Prefix, Env: first.Env}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("after add = %#v", got)
	}
	updated := LaunchPreset{Name: "proton", Options: "-foo", Prefix: "mangohud", Env: "A=1"}
	if err := s.AddLaunchPreset(GameStardew, updated); err != nil {
		t.Fatal(err)
	}
	got = s.ListLaunchPresets(GameStardew)
	if len(got) != 1 || got[0].Options != "-foo" || got[0].Name != "proton" {
		t.Fatalf("replace same name = %#v", got)
	}
	if err := s.AddLaunchPreset(GameStardew, LaunchPreset{Name: "Direct"}); err != nil {
		t.Fatal(err)
	}
	if got = s.ListLaunchPresets(GameStardew); len(got) != 2 {
		t.Fatalf("two presets = %#v", got)
	}
	if err := s.RemoveLaunchPreset(GameStardew, "PROTON"); err != nil {
		t.Fatal(err)
	}
	got = s.ListLaunchPresets(GameStardew)
	if len(got) != 1 || got[0].Name != "Direct" {
		t.Fatalf("after remove = %#v", got)
	}
	if err := s.RemoveLaunchPreset(GameStardew, "missing"); err == nil {
		t.Fatal("expected missing preset error")
	}
	if err := s.AddLaunchPreset(GameStardew, LaunchPreset{Name: "  "}); err == nil {
		t.Fatal("expected empty name error")
	}
}

func TestLaunchPresetPersists(t *testing.T) {
	s, _ := open(t)
	preset := LaunchPreset{Name: "Keep", Options: "-x", Prefix: "p", Env: "E=1"}
	if err := s.AddLaunchPreset(GameStardew, preset); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	got := s2.ListLaunchPresets(GameStardew)
	if len(got) != 1 || !reflect.DeepEqual(got[0], preset) {
		t.Fatalf("reloaded = %#v", got)
	}
}
