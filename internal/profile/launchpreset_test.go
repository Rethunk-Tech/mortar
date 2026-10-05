package profile

import "testing"

func TestResolvePreset(t *testing.T) {
	t.Parallel()
	p := Profile{
		Name: "Farm", LaunchOptions: "--a", LaunchPrefix: "gamemoderun", LaunchEnv: "X=1",
		LaunchPresets: []LaunchPreset{
			{ID: "p1", Name: "Debug", LaunchOptions: "--b", ShowConsole: "true"},
			{ID: "p2", Name: "Quiet"},
		},
	}
	base, err := p.ResolvePreset("")
	if err != nil || base.Name != BasePresetName || base.Options != "--a" || base.Prefix != "gamemoderun" || base.Env != "X=1" {
		t.Fatalf("empty name with no default = %+v, %v", base, err)
	}
	p.DefaultLaunchPreset = "p2"
	if got, _ := p.ResolvePreset(""); got.Name != "Quiet" || got.Options != "" {
		t.Fatalf("default preset = %+v", got)
	}
	if got, _ := p.ResolvePreset("debug"); got.Options != "--b" || got.ShowConsole != "true" {
		t.Fatalf("by name ignoring case = %+v", got)
	}
	if got, _ := p.ResolvePreset("p1"); got.Name != "Debug" {
		t.Fatalf("by id = %+v", got)
	}
	if got, _ := p.ResolvePreset("standard"); got.Options != "--a" {
		t.Fatalf("base by name = %+v", got)
	}
	if _, err := p.ResolvePreset("nope"); err == nil {
		t.Fatal("unknown preset accepted")
	}
	p.DefaultLaunchPreset = "gone"
	if got, _ := p.ResolvePreset(""); got.Name != BasePresetName {
		t.Fatalf("dangling default = %+v", got)
	}
}

func TestLaunchSpecOverrides(t *testing.T) {
	t.Parallel()
	if got := (LaunchSpec{}).Overrides(nil); got != nil {
		t.Fatalf("no console choice = %v", got)
	}
	base := map[string]string{"a": "b"}
	got := LaunchSpec{ShowConsole: "false"}.Overrides(base)
	if got[showConsoleKey] != "false" || got["a"] != "b" || len(base) != 1 {
		t.Fatalf("overrides = %v base = %v", got, base)
	}
}

func TestSetLaunchPresets(t *testing.T) {
	t.Parallel()
	s := overrideStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	for name, presets := range map[string][]LaunchPreset{
		"empty name":     {{Name: " "}},
		"base name":      {{Name: "standard"}},
		"duplicate name": {{Name: "A"}, {Name: "a"}},
		"denied flag":    {{Name: "A", LaunchOptions: "--no-terminal"}},
		"bad env":        {{Name: "A", LaunchEnv: "nope"}},
		"bad console":    {{Name: "A", ShowConsole: "maybe"}},
	} {
		if _, err := s.SetLaunchPresets("stardew", p.ID, presets, ""); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	got, err := s.SetLaunchPresets("stardew", p.ID, []LaunchPreset{{Name: " Debug ", LaunchOptions: "--x"}}, "")
	if err != nil || len(got.LaunchPresets) != 1 || got.LaunchPresets[0].ID == "" || got.LaunchPresets[0].Name != "Debug" {
		t.Fatalf("saved = %+v, %v", got.LaunchPresets, err)
	}
	id := got.LaunchPresets[0].ID
	if _, err := s.SetDefaultLaunchPreset("stardew", p.ID, "missing"); err == nil {
		t.Fatal("missing default accepted")
	}
	if _, err := s.SetDefaultLaunchPreset("stardew", p.ID, id); err != nil {
		t.Fatal(err)
	}
	spec, err := s.LaunchSpec("stardew", p.ID, "")
	if err != nil || spec.Name != "Debug" || spec.Options != "--x" {
		t.Fatalf("spec = %+v, %v", spec, err)
	}
	if _, err := s.SetLaunchPresets("stardew", p.ID, nil, id); err == nil {
		t.Fatal("default of a removed preset accepted")
	}
}

func TestAddLaunchPresetUniquifiesName(t *testing.T) {
	t.Parallel()
	s := overrideStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	var got Profile
	for _, name := range []string{"Debug", "debug", "Standard"} {
		if got, err = s.AddLaunchPreset("stardew", p.ID, LaunchPreset{ID: "x", Name: name, LaunchOptions: "--x"}); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{}
	for _, preset := range got.LaunchPresets {
		names = append(names, preset.Name)
		if preset.ID == "" || preset.ID == "x" {
			t.Fatalf("id not fresh: %+v", preset)
		}
	}
	if len(names) != 3 || names[0] != "Debug" || names[1] != "debug 2" || names[2] != "Standard 2" {
		t.Fatalf("names = %v", names)
	}
}
